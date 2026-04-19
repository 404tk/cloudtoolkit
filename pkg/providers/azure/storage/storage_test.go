package storage

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

func TestDriverGetStoragesFollowsPagination(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tenant/oauth2/v2.0/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case "/subscriptions/sub-1/providers/Microsoft.Storage/storageAccounts":
			_, _ = w.Write([]byte(`{"value":[{"id":"/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Storage/storageAccounts/acct-1","name":"acct-1","location":"eastasia"}]}`))
		case "/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Storage/storageAccounts/acct-1/blobServices":
			_, _ = w.Write([]byte(`{"value":[{"name":"default"}]}`))
		case "/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Storage/storageAccounts/acct-1/blobServices/default/containers":
			if r.URL.Query().Get("page") == "2" {
				_, _ = w.Write([]byte(`{"value":[{"name":"container-2"}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"value":[{"name":"container-1"}],"nextLink":"` + server.URL + `/subscriptions/sub-1/resourceGroups/rg-1/providers/Microsoft.Storage/storageAccounts/acct-1/blobServices/default/containers?page=2"}`))
		default:
			t.Fatalf("unexpected path: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
	}))
	defer server.Close()

	httpClient := server.Client()
	httpClient.Transport = tokenRewriteTransport{base: httpClient.Transport, target: mustParseURL(t, server.URL)}
	ts := auth.NewTokenSource(auth.New("client", "secret", "tenant", "", auth.CloudPublic), httpClient)
	client := azapi.NewClient(ts, cloud.For(auth.CloudPublic), azapi.WithHTTPClient(httpClient), azapi.WithBaseURL(server.URL))

	driver := &Driver{
		Client:          client,
		SubscriptionIDs: []string{"sub-1"},
	}
	got, err := driver.GetStorages(context.Background())
	if err != nil {
		t.Fatalf("GetStorages failed: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("unexpected storage count: %d", len(got))
	}
	if got[0].BucketName != "default(Blob Service)" || got[1].BucketName != "container-1(Blob Container)" || got[2].BucketName != "container-2(Blob Container)" {
		t.Fatalf("unexpected storages: %+v", got)
	}
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
