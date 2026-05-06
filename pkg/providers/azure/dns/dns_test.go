package dns

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	azapi "github.com/404tk/cloudtoolkit/pkg/providers/azure/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/cloud"
)

const tokenStub = `{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`

func newDNSTestDriver(t *testing.T, server *httptest.Server, subs []string) *Driver {
	t.Helper()
	httpClient := server.Client()
	httpClient.Transport = tokenRewriteTransport{base: httpClient.Transport, target: mustParseURL(t, server.URL)}
	ts := auth.NewTokenSource(auth.New("client", "secret", "tenant", "", auth.CloudPublic), httpClient)
	client := azapi.NewClient(ts, cloud.For(auth.CloudPublic), azapi.WithHTTPClient(httpClient), azapi.WithBaseURL(server.URL))
	return &Driver{Client: client, SubscriptionIDs: subs}
}

type tokenRewriteTransport struct {
	base   http.RoundTripper
	target *url.URL
}

func (rt tokenRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == "login.microsoftonline.com" {
		clone := req.Clone(req.Context())
		clone.URL.Scheme = rt.target.Scheme
		clone.URL.Host = rt.target.Host
		clone.Host = rt.target.Host
		return rt.base.RoundTrip(clone)
	}
	return rt.base.RoundTrip(req)
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	return u
}

const sampleDNSZones = `{"value":[
  {"id":"/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Network/dnsZones/ctk-demo.example.com",
   "name":"ctk-demo.example.com",
   "type":"Microsoft.Network/dnszones",
   "location":"global",
   "properties":{"numberOfRecordSets":4,"zoneType":"Public"}}
]}`

const sampleDNSRecords = `{"value":[
  {"id":"/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Network/dnsZones/ctk-demo.example.com/A/@",
   "name":"@","type":"Microsoft.Network/dnszones/A",
   "properties":{"TTL":300,"fqdn":"ctk-demo.example.com.","ARecords":[{"ipv4Address":"203.0.113.10"},{"ipv4Address":"203.0.113.11"}]}},
  {"id":"/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Network/dnsZones/ctk-demo.example.com/CNAME/www",
   "name":"www","type":"Microsoft.Network/dnszones/CNAME",
   "properties":{"TTL":60,"fqdn":"www.ctk-demo.example.com.","CNAMERecord":{"cname":"ctk-demo.example.com."}}},
  {"id":"/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Network/dnsZones/ctk-demo.example.com/NS/@",
   "name":"@","type":"Microsoft.Network/dnszones/NS",
   "properties":{"TTL":172800,"fqdn":"ctk-demo.example.com.","NSRecords":[{"nsdname":"ns1-01.azure-dns.com."}]}},
  {"id":"/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Network/dnsZones/ctk-demo.example.com/MX/mail",
   "name":"mail","type":"Microsoft.Network/dnszones/MX",
   "properties":{"TTL":300,"fqdn":"mail.ctk-demo.example.com.","MXRecords":[{"preference":10,"exchange":"mx1.ctk-demo.example.com."}]}}
]}`

func dnsTestServer(t *testing.T, recordHandler func(http.ResponseWriter, *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/tenant/oauth2/v2.0/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(tokenStub))
		case r.URL.Path == "/subscriptions/sub-1/providers/Microsoft.Network/dnsZones":
			_, _ = w.Write([]byte(sampleDNSZones))
		case strings.HasSuffix(r.URL.Path, "/recordsets"):
			recordHandler(w, r)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
}

func TestGetDomainsListsZonesAndRecordSets(t *testing.T) {
	server := dnsTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleDNSRecords))
	})
	defer server.Close()

	driver := newDNSTestDriver(t, server, []string{"sub-1"})
	domains, err := driver.GetDomains(context.Background())
	if err != nil {
		t.Fatalf("GetDomains: %v", err)
	}
	if len(domains) != 1 {
		t.Fatalf("expected 1 domain, got %d", len(domains))
	}
	d := domains[0]
	if d.DomainName != "ctk-demo.example.com" {
		t.Errorf("unexpected zone name: %q", d.DomainName)
	}
	// 2 A + 1 CNAME + 1 MX = 4 records (NS filtered)
	if len(d.Records) != 4 {
		t.Fatalf("expected 4 records, got %d (%+v)", len(d.Records), d.Records)
	}
	var sawA, sawCNAME, sawMX bool
	for _, rec := range d.Records {
		switch rec.Type {
		case "A":
			if rec.Value == "203.0.113.10" || rec.Value == "203.0.113.11" {
				sawA = true
			}
			if rec.RR != "ctk-demo.example.com" {
				t.Errorf("A record RR should be FQDN, got %q", rec.RR)
			}
		case "CNAME":
			sawCNAME = true
			if rec.Value != "ctk-demo.example.com" {
				t.Errorf("CNAME value should strip trailing dot, got %q", rec.Value)
			}
		case "MX":
			sawMX = true
			if rec.Value != "10 mx1.ctk-demo.example.com" {
				t.Errorf("MX value mismatch: %q", rec.Value)
			}
		case "NS":
			t.Errorf("NS records should be filtered, found %+v", rec)
		}
	}
	if !sawA || !sawCNAME || !sawMX {
		t.Errorf("missing record kinds: A=%v CNAME=%v MX=%v", sawA, sawCNAME, sawMX)
	}
}

func TestGetDomainsContinuesPastRecordSetFailure(t *testing.T) {
	server := dnsTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"code":"AuthorizationFailed","message":"forbidden"}}`, http.StatusForbidden)
	})
	defer server.Close()

	driver := newDNSTestDriver(t, server, []string{"sub-1"})
	domains, err := driver.GetDomains(context.Background())
	if err != nil {
		t.Fatalf("GetDomains returned fatal err on per-zone failure: %v", err)
	}
	if len(domains) != 1 {
		t.Fatalf("expected zone preserved on rrset error, got %d domains", len(domains))
	}
	if len(domains[0].Records) != 0 {
		t.Errorf("expected empty records on denied zone, got %+v", domains[0].Records)
	}
}

func TestGetDomainsPropagatesZoneListFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tenant/oauth2/v2.0/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(tokenStub))
		case "/subscriptions/sub-1/providers/Microsoft.Network/dnsZones":
			http.Error(w, `{"error":{"code":"InvalidAuthenticationToken","message":"bad creds"}}`, http.StatusUnauthorized)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := newDNSTestDriver(t, server, []string{"sub-1"})
	_, err := driver.GetDomains(context.Background())
	if err == nil {
		t.Fatalf("expected error when zone list fails")
	}
	if !strings.Contains(err.Error(), "InvalidAuthenticationToken") {
		t.Errorf("expected error to mention InvalidAuthenticationToken; got: %v", err)
	}
}
