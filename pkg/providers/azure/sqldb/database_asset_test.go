package sqldb

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

func newAssetDriver(t *testing.T, server *httptest.Server, subs []string) *Driver {
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

const sampleSQLServers = `{"value":[
  {"id":"/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Sql/servers/ctk-prod-sql","name":"ctk-prod-sql","location":"eastus",
   "properties":{"administratorLogin":"sqladmin","version":"12.0","state":"Ready","fullyQualifiedDomainName":"ctk-prod-sql.database.windows.net"}},
  {"id":"/subscriptions/sub-1/resourceGroups/rg-2/providers/Microsoft.Sql/servers/ctk-stage-sql","name":"ctk-stage-sql","location":"westus2",
   "properties":{"version":"12.0","state":"Ready"}}
]}`

func TestGetDatabasesListsServers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tenant/oauth2/v2.0/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(tokenStub))
		case "/subscriptions/sub-1/providers/Microsoft.Sql/servers":
			_, _ = w.Write([]byte(sampleSQLServers))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := newAssetDriver(t, server, []string{"sub-1"})
	dbs, err := driver.GetDatabases(context.Background())
	if err != nil {
		t.Fatalf("GetDatabases: %v", err)
	}
	if len(dbs) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(dbs))
	}
	if dbs[0].InstanceId != "ctk-prod-sql" || dbs[0].Region != "eastus" {
		t.Errorf("unexpected first db: %+v", dbs[0])
	}
	if dbs[0].NetworkType != "Public" || dbs[1].NetworkType != "Private" {
		t.Errorf("network types mismatch: %+v / %+v", dbs[0].NetworkType, dbs[1].NetworkType)
	}
	if !strings.Contains(dbs[0].Address, "database.windows.net") {
		t.Errorf("expected FQDN in address, got %q", dbs[0].Address)
	}
}

func TestGetDatabasesPropagatesAuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tenant/oauth2/v2.0/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(tokenStub))
		case "/subscriptions/sub-1/providers/Microsoft.Sql/servers":
			http.Error(w, `{"error":{"code":"InvalidAuthenticationToken","message":"bad creds"}}`, http.StatusUnauthorized)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := newAssetDriver(t, server, []string{"sub-1"})
	_, err := driver.GetDatabases(context.Background())
	if err == nil {
		t.Fatal("expected error when listing SQL servers fails")
	}
	if !strings.Contains(err.Error(), "InvalidAuthenticationToken") {
		t.Errorf("expected InvalidAuthenticationToken in err, got %v", err)
	}
}

func TestGetDatabasesHandlesEmptyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tenant/oauth2/v2.0/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(tokenStub))
		case "/subscriptions/sub-1/providers/Microsoft.Sql/servers":
			_, _ = w.Write([]byte(`{"value":[]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := newAssetDriver(t, server, []string{"sub-1"})
	dbs, err := driver.GetDatabases(context.Background())
	if err != nil {
		t.Fatalf("GetDatabases: %v", err)
	}
	if len(dbs) != 0 {
		t.Errorf("expected 0 servers, got %d", len(dbs))
	}
}
