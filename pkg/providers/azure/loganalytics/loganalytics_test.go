package loganalytics

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

func newTestDriver(t *testing.T, server *httptest.Server, subs []string) *Driver {
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

const sampleWorkspaces = `{"value":[
  {"id":"/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.OperationalInsights/workspaces/ctk-prod",
   "name":"ctk-prod","type":"Microsoft.OperationalInsights/workspaces","location":"eastus",
   "properties":{"customerId":"cus-1","provisioningState":"Succeeded","createdDate":"2026-01-01","modifiedDate":"2026-04-15","retentionInDays":30,"sku":{"name":"PerGB2018"}}},
  {"id":"/subscriptions/sub-1/resourceGroups/rg-2/providers/Microsoft.OperationalInsights/workspaces/ctk-dev",
   "name":"ctk-dev","type":"Microsoft.OperationalInsights/workspaces","location":"westus2",
   "properties":{"customerId":"cus-2","provisioningState":"Succeeded","createdDate":"2026-02-01","modifiedDate":"2026-04-20","retentionInDays":7}}
]}`

func TestGetLogsListsWorkspaces(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tenant/oauth2/v2.0/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(tokenStub))
		case "/subscriptions/sub-1/providers/Microsoft.OperationalInsights/workspaces":
			_, _ = w.Write([]byte(sampleWorkspaces))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := newTestDriver(t, server, []string{"sub-1"})
	logs, err := driver.GetLogs(context.Background())
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(logs))
	}
	if logs[0].ProjectName != "ctk-prod" || logs[0].Region != "eastus" {
		t.Errorf("unexpected first workspace: %+v", logs[0])
	}
	if !strings.Contains(logs[0].Description, "retention=30") || !strings.Contains(logs[0].Description, "sku=PerGB2018") {
		t.Errorf("description should include retention+sku, got %q", logs[0].Description)
	}
	if logs[1].LastModifyTime != "2026-04-20" {
		t.Errorf("expected modified date carry-through, got %q", logs[1].LastModifyTime)
	}
}

func TestGetLogsPropagatesAuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tenant/oauth2/v2.0/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(tokenStub))
		case "/subscriptions/sub-1/providers/Microsoft.OperationalInsights/workspaces":
			http.Error(w, `{"error":{"code":"InvalidAuthenticationToken","message":"bad"}}`, http.StatusUnauthorized)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := newTestDriver(t, server, []string{"sub-1"})
	_, err := driver.GetLogs(context.Background())
	if err == nil {
		t.Fatal("expected error when listing workspaces fails")
	}
	if !strings.Contains(err.Error(), "InvalidAuthenticationToken") {
		t.Errorf("expected InvalidAuthenticationToken in err, got %v", err)
	}
}

func TestGetLogsHandlesEmptyWorkspaceList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tenant/oauth2/v2.0/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(tokenStub))
		case "/subscriptions/sub-1/providers/Microsoft.OperationalInsights/workspaces":
			_, _ = w.Write([]byte(`{"value":[]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := newTestDriver(t, server, []string{"sub-1"})
	logs, err := driver.GetLogs(context.Background())
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 workspaces, got %d", len(logs))
	}
}
