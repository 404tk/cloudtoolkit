package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/azure/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/cloud"
)

func TestPagerAllFollowsNextLink(t *testing.T) {
	var tokenCalls int
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/tenant/oauth2/v2.0/token":
			tokenCalls++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case r.URL.Path == "/subscriptions" && r.URL.Query().Get("page") == "2":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"value":[{"subscriptionId":"sub-2","displayName":"two"}]}`))
		case r.URL.Path == "/subscriptions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"value":[{"subscriptionId":"sub-1","displayName":"one"}],"nextLink":"` + server.URL + `/subscriptions?page=2"}`))
		default:
			t.Fatalf("unexpected path: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
	}))
	defer server.Close()

	httpClient := server.Client()
	httpClient.Transport = rewriteTokenHostTransport(t, httpClient.Transport, server.URL)
	ts := auth.NewTokenSource(auth.New("client", "secret", "tenant", "", auth.CloudPublic), httpClient)
	client := NewClient(ts, cloud.For(auth.CloudPublic), WithHTTPClient(httpClient), WithBaseURL(server.URL))

	pager := NewPager[Subscription](client, Request{
		Method:     http.MethodGet,
		Path:       "/subscriptions",
		Query:      url.Values{"api-version": {SubscriptionsAPIVersion}},
		Idempotent: true,
	})
	items, err := pager.All(context.Background())
	if err != nil {
		t.Fatalf("pager failed: %v", err)
	}
	if len(items) != 2 || items[0].SubscriptionID != "sub-1" || items[1].SubscriptionID != "sub-2" {
		t.Fatalf("unexpected items: %+v", items)
	}
	if tokenCalls != 1 {
		t.Fatalf("unexpected token calls: %d", tokenCalls)
	}
}
