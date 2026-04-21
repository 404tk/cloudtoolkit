package dns

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/internal/testutil"
)

func TestGetDomainsFiltersRecordsAndFallsBackToZoneName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case "/dns/v1/projects/proj-1/managedZones":
			_, _ = w.Write([]byte(`{"managedZones":[{"name":"zone-a","dnsName":"example.com."},{"name":"zone-b","dnsName":""}]}`))
		case "/dns/v1/projects/proj-1/managedZones/zone-a/rrsets":
			_, _ = w.Write([]byte(`{"rrsets":[{"name":"example.com.","type":"A","rrdatas":["1.1.1.1"]},{"name":"www.example.com.","type":"CNAME","rrdatas":["example.com."]},{"name":"txt.example.com.","type":"TXT","rrdatas":["ignored"]}]}`))
		case "/dns/v1/projects/proj-1/managedZones/zone-b/rrsets":
			_, _ = w.Write([]byte(`{"rrsets":[{"name":"ipv6.example.com.","type":"AAAA","rrdatas":["2001:db8::1"]}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{
		Projects: []string{"proj-1"},
		Client:   newTestClient(t, server),
	}
	domains, err := driver.GetDomains(context.Background())
	if err != nil {
		t.Fatalf("GetDomains() error = %v", err)
	}
	if len(domains) != 2 {
		t.Fatalf("unexpected domain count: %d", len(domains))
	}
	if domains[0].DomainName != "example.com." || len(domains[0].Records) != 2 {
		t.Fatalf("unexpected first domain: %+v", domains[0])
	}
	if domains[1].DomainName != "zone-b" || len(domains[1].Records) != 1 || domains[1].Records[0].Type != "AAAA" {
		t.Fatalf("unexpected second domain: %+v", domains[1])
	}
}

func newTestClient(t *testing.T, server *httptest.Server) *api.Client {
	t.Helper()
	httpClient := server.Client()
	transport, err := testutil.RewriteHostsTransport(httpClient.Transport, server.URL, "dns.googleapis.com")
	if err != nil {
		t.Fatalf("RewriteHostsTransport() error = %v", err)
	}
	httpClient.Transport = transport

	ts := auth.NewTokenSource(auth.Credential{
		Type:          "service_account",
		ProjectID:     "proj-1",
		PrivateKeyID:  "kid-1",
		PrivateKeyPEM: testutil.PKCS8PrivateKeyPEM,
		ClientEmail:   "demo@example.com",
		TokenURI:      server.URL + "/token",
		Scopes:        []string{auth.DefaultScope},
	}, httpClient)
	return api.NewClient(ts, api.WithHTTPClient(httpClient))
}
