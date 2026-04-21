package auth

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/internal/testutil"
)

func TestTokenSourceCachesAndRefreshes(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if got := r.URL.Path; got != "/token" {
			t.Fatalf("unexpected token path: %s", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		if got := r.Form.Get("grant_type"); got != "urn:ietf:params:oauth:grant-type:jwt-bearer" {
			t.Fatalf("unexpected grant_type: %s", got)
		}
		if got := r.Form.Get("assertion"); got == "" {
			t.Fatal("expected non-empty assertion")
		}
		token := "token-1"
		if requests.Load() > 1 {
			token = "token-2"
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"` + token + `","expires_in":120,"token_type":"Bearer"}`))
	}))
	defer server.Close()

	now := time.Unix(1700000000, 0)
	ts := NewTokenSource(newTestCredential(server.URL+"/token"), server.Client())
	ts.clock = func() time.Time { return now }

	first, err := ts.Token(context.Background())
	if err != nil {
		t.Fatalf("first Token() error = %v", err)
	}
	second, err := ts.Token(context.Background())
	if err != nil {
		t.Fatalf("second Token() error = %v", err)
	}
	if first.AccessToken != "token-1" || second.AccessToken != "token-1" {
		t.Fatalf("unexpected tokens: %+v %+v", first, second)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("expected one request before refresh, got %d", got)
	}

	now = now.Add(91 * time.Second)
	refreshed, err := ts.Token(context.Background())
	if err != nil {
		t.Fatalf("refresh Token() error = %v", err)
	}
	if refreshed.AccessToken != "token-2" {
		t.Fatalf("unexpected refreshed token: %+v", refreshed)
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("expected refresh request, got %d", got)
	}
}

func TestTokenSourceConcurrentFetchSingleFlight(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		time.Sleep(10 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"shared-token","expires_in":120,"token_type":"Bearer"}`))
	}))
	defer server.Close()

	ts := NewTokenSource(newTestCredential(server.URL+"/token"), server.Client())
	ts.clock = func() time.Time { return time.Unix(1700000000, 0) }

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, err := ts.Token(context.Background())
			if err != nil {
				t.Errorf("Token() error = %v", err)
				return
			}
			if token.AccessToken != "shared-token" {
				t.Errorf("unexpected token: %s", token.AccessToken)
			}
		}()
	}
	wg.Wait()

	if got := requests.Load(); got != 1 {
		t.Fatalf("expected one token request, got %d", got)
	}
}

func TestTokenSourceErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"service account disabled"}`))
	}))
	defer server.Close()

	ts := NewTokenSource(newTestCredential(server.URL+"/token"), server.Client())
	_, err := ts.Token(context.Background())
	if err == nil {
		t.Fatal("expected token error")
	}
	if got := err.Error(); got != "gcp oauth2: invalid_grant: service account disabled" {
		t.Fatalf("unexpected error: %s", got)
	}
}

func newTestCredential(tokenURL string) Credential {
	return Credential{
		Type:          "service_account",
		ProjectID:     "demo-project",
		PrivateKeyID:  "kid-1",
		PrivateKeyPEM: testutil.PKCS8PrivateKeyPEM,
		ClientEmail:   "demo@example.com",
		TokenURI:      tokenURL,
		Scopes:        []string{DefaultScope},
	}
}

func TestTokenSourceRequestBodyIsFormEncoded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Fatalf("unexpected content type: %s", got)
		}
		body, err := url.ParseQuery(readBody(t, r))
		if err != nil {
			t.Fatalf("ParseQuery() error = %v", err)
		}
		if body.Get("grant_type") != "urn:ietf:params:oauth:grant-type:jwt-bearer" {
			t.Fatalf("unexpected grant_type: %s", body.Get("grant_type"))
		}
		if body.Get("assertion") == "" {
			t.Fatal("expected assertion")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
	}))
	defer server.Close()

	ts := NewTokenSource(newTestCredential(server.URL+"/token"), server.Client())
	if _, err := ts.Token(context.Background()); err != nil {
		t.Fatalf("Token() error = %v", err)
	}
}

func readBody(t *testing.T, r *http.Request) string {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(body)
}
