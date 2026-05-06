package billing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
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

const sampleQueryResponse = `{"id":"q1","name":"q","type":"Microsoft.CostManagement/query","properties":{
  "columns":[{"name":"Cost","type":"Number"},{"name":"Currency","type":"String"}],
  "rows":[[42.789,"USD"]]
}}`

func TestQueryAccountBalanceLogsCurrentMonthSpend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tenant/oauth2/v2.0/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(tokenStub))
		case "/subscriptions/sub-1/providers/Microsoft.CostManagement/query":
			if got := r.Method; got != http.MethodPost {
				t.Errorf("expected POST, got %s", got)
			}
			_, _ = w.Write([]byte(sampleQueryResponse))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := newTestDriver(t, server, []string{"sub-1"})
	driver.QueryAccountBalance(context.Background())
}

func TestQueryAccountBalanceSwallowsAccessDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tenant/oauth2/v2.0/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(tokenStub))
		case "/subscriptions/sub-1/providers/Microsoft.CostManagement/query":
			http.Error(w, `{"error":{"code":"AuthorizationFailed","message":"forbidden"}}`, http.StatusForbidden)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := newTestDriver(t, server, []string{"sub-1"})
	// Should not panic / propagate error — balance is best-effort.
	driver.QueryAccountBalance(context.Background())
}

func TestParseFirstRowReadsCurrencyAndAmount(t *testing.T) {
	resp := azapi.CostManagementQueryResponse{
		Properties: azapi.CostManagementQueryProperties{
			Columns: []azapi.CostManagementColumn{
				{Name: "Cost", Type: "Number"},
				{Name: "BillingCurrency", Type: "String"},
			},
			Rows: [][]any{{12.5, "EUR"}},
		},
	}
	amount, currency, err := parseFirstRow(resp)
	if err != nil {
		t.Fatalf("parseFirstRow: %v", err)
	}
	if amount != "12.5" {
		t.Errorf("expected trimmed amount, got %q", amount)
	}
	if currency != "EUR" {
		t.Errorf("expected EUR currency, got %q", currency)
	}
}
