package billing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/internal/testutil"
)

func newClient(t *testing.T, server *httptest.Server) *api.Client {
	t.Helper()
	httpClient := server.Client()
	transport, err := testutil.RewriteHostsTransport(httpClient.Transport, server.URL, "cloudbilling.googleapis.com")
	if err != nil {
		t.Fatalf("RewriteHostsTransport: %v", err)
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

func TestQueryAccountBalanceLogsVisibleAccounts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_, _ = w.Write([]byte(`{"access_token":"demo","token_type":"Bearer","expires_in":3600}`))
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/v1/billingAccounts") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"billingAccounts":[
  {"name":"billingAccounts/01-AAA-BBB","displayName":"Production","open":true},
  {"name":"billingAccounts/02-CCC-DDD","displayName":"Sandbox","open":false}
]}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newClient(t, server)}
	driver.QueryAccountBalance(context.Background())
}

func TestQueryAccountBalanceSwallowsAccessDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_, _ = w.Write([]byte(`{"access_token":"demo","token_type":"Bearer","expires_in":3600}`))
			return
		}
		http.Error(w, `{"error":{"code":403,"message":"forbidden","status":"PERMISSION_DENIED"}}`, http.StatusForbidden)
	}))
	defer server.Close()

	driver := &Driver{Client: newClient(t, server)}
	driver.QueryAccountBalance(context.Background())
}

func TestListPaginates(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_, _ = w.Write([]byte(`{"access_token":"demo","token_type":"Bearer","expires_in":3600}`))
			return
		}
		calls++
		if calls == 1 {
			_, _ = w.Write([]byte(`{"billingAccounts":[{"name":"billingAccounts/01","open":true}],"nextPageToken":"p2"}`))
			return
		}
		_, _ = w.Write([]byte(`{"billingAccounts":[{"name":"billingAccounts/02","open":true}]}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newClient(t, server)}
	accounts, err := driver.list(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(accounts) != 2 {
		t.Errorf("expected 2 accounts after pagination, got %d", len(accounts))
	}
	if calls != 2 {
		t.Errorf("expected 2 list calls, got %d", calls)
	}
}
