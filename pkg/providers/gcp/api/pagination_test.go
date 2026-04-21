package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/internal/testutil"
)

type namedItem struct {
	Name string `json:"name"`
}

func TestPagerAllFollowsNextPageToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case "/items":
			switch r.URL.Query().Get("pageToken") {
			case "":
				if got := r.URL.Query().Get("maxResults"); got != "500" {
					t.Fatalf("unexpected maxResults: %s", got)
				}
				_, _ = w.Write([]byte(`{"items":[{"name":"one"}],"nextPageToken":"page-2"}`))
			case "page-2":
				_, _ = w.Write([]byte(`{"items":[{"name":"two"}]}`))
			default:
				t.Fatalf("unexpected pageToken: %s", r.URL.Query().Get("pageToken"))
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	pager := NewPager[namedItem](newTestAPIClient(server), Request{
		Method:     http.MethodGet,
		BaseURL:    server.URL,
		Path:       "/items",
		Idempotent: true,
	}, "items")
	items, err := pager.All(context.Background())
	if err != nil {
		t.Fatalf("All() error = %v", err)
	}
	if len(items) != 2 || items[0].Name != "one" || items[1].Name != "two" {
		t.Fatalf("unexpected items: %+v", items)
	}
}

func TestPagerAllHandlesAccountsAndEmptyPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case "/accounts":
			_, _ = w.Write([]byte(`{"accounts":[]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	pager := NewPager[namedItem](newTestAPIClient(server), Request{
		Method:     http.MethodGet,
		BaseURL:    server.URL,
		Path:       "/accounts",
		Query:      url.Values{"pageSize": {"100"}},
		Idempotent: true,
	}, "accounts")
	items, err := pager.All(context.Background())
	if err != nil {
		t.Fatalf("All() error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty items, got %+v", items)
	}
}

func TestPagerClonesQueryPerCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case "/items":
			switch r.URL.Query().Get("pageToken") {
			case "":
				_, _ = w.Write([]byte(`{"items":[{"name":"one"}],"nextPageToken":"page-2"}`))
			case "page-2":
				_, _ = w.Write([]byte(`{"items":[{"name":"two"}]}`))
			default:
				t.Fatalf("unexpected pageToken: %s", r.URL.Query().Get("pageToken"))
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := newTestAPIClient(server)
	initial := Request{
		Method:     http.MethodGet,
		BaseURL:    server.URL,
		Path:       "/items",
		Query:      url.Values{"filter": {"x"}},
		Idempotent: true,
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := NewPager[namedItem](client, initial, "items").All(context.Background())
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("All() error = %v", err)
		}
	}
	if got := initial.Query.Get("pageToken"); got != "" {
		t.Fatalf("expected original query to stay clean, got pageToken=%s", got)
	}
}

func newTestAPIClient(server *httptest.Server) *Client {
	httpClient := server.Client()
	ts := auth.NewTokenSource(auth.Credential{
		Type:          "service_account",
		ProjectID:     "demo-project",
		PrivateKeyID:  "kid-1",
		PrivateKeyPEM: testutil.PKCS8PrivateKeyPEM,
		ClientEmail:   "demo@example.com",
		TokenURI:      server.URL + "/token",
		Scopes:        []string{auth.DefaultScope},
	}, httpClient)
	return NewClient(ts, WithHTTPClient(httpClient))
}
