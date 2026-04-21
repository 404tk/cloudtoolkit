package compute

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/internal/testutil"
)

func TestGetResourceMapsInstancesAndHandlesPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case "/compute/v1/projects/proj-1/zones":
			_, _ = w.Write([]byte(`{"items":[{"name":"zone-a"},{"name":"zone-b"}]}`))
		case "/compute/v1/projects/proj-1/zones/zone-a/instances":
			switch r.URL.Query().Get("pageToken") {
			case "":
				_, _ = w.Write([]byte(`{"items":[{"hostname":"vm-1","zone":"https://www.googleapis.com/compute/v1/projects/proj-1/zones/zone-a","networkInterfaces":[{"networkIP":"10.0.0.1"},{"networkIP":"10.0.0.2","accessConfigs":[{"natIP":"1.1.1.1"},{"natIP":"1.1.1.2"}]}]}],"nextPageToken":"page-2"}`))
			case "page-2":
				_, _ = w.Write([]byte(`{"items":[{"hostname":"vm-2","zone":"https://www.googleapis.com/compute/v1/projects/proj-1/zones/zone-a","networkInterfaces":[{"networkIP":"10.0.1.9"}]}]}`))
			default:
				t.Fatalf("unexpected pageToken: %s", r.URL.Query().Get("pageToken"))
			}
		case "/compute/v1/projects/proj-1/zones/zone-b/instances":
			_, _ = w.Write([]byte(`{"items":[{"hostname":"vm-3","zone":"https://www.googleapis.com/compute/v1/projects/proj-1/zones/zone-b","networkInterfaces":[{"networkIP":"10.0.2.5","accessConfigs":[{"natIP":"2.2.2.2"}]}]}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{
		Projects: []string{"proj-1"},
		Client:   newTestClient(t, server),
	}
	hosts, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}
	if len(hosts) != 3 {
		t.Fatalf("unexpected host count: %d", len(hosts))
	}
	if hosts[0].HostName != "vm-1" || hosts[0].PrivateIpv4 != "10.0.0.2" || hosts[0].PublicIPv4 != "1.1.1.1" || !hosts[0].Public {
		t.Fatalf("unexpected first host: %+v", hosts[0])
	}
	if hosts[1].HostName != "vm-2" || hosts[1].Public || hosts[1].PublicIPv4 != "" || hosts[1].PrivateIpv4 != "10.0.1.9" {
		t.Fatalf("unexpected second host: %+v", hosts[1])
	}
	if hosts[2].Region != "https://www.googleapis.com/compute/v1/projects/proj-1/zones/zone-b" {
		t.Fatalf("unexpected third host region: %+v", hosts[2])
	}
}

func newTestClient(t *testing.T, server *httptest.Server) *api.Client {
	t.Helper()
	httpClient := server.Client()
	transport, err := testutil.RewriteHostsTransport(httpClient.Transport, server.URL, "compute.googleapis.com")
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
