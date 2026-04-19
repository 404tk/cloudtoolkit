package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTokenSourceCachesAndRefreshes(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if got := r.URL.Path; got != "/tenant/oauth2/v2.0/token" {
			t.Fatalf("unexpected token path: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token-1","expires_in":120,"token_type":"Bearer"}`))
	}))
	defer server.Close()

	now := time.Unix(1700000000, 0)
	ts := NewTokenSource(New("client", "secret", "tenant", "", CloudPublic), server.Client())
	ts.tokenURL = server.URL + "/tenant/oauth2/v2.0/token"
	ts.clock = func() time.Time { return now }

	first, err := ts.Token(context.Background())
	if err != nil {
		t.Fatalf("first token failed: %v", err)
	}
	second, err := ts.Token(context.Background())
	if err != nil {
		t.Fatalf("second token failed: %v", err)
	}
	if first.AccessToken != "token-1" || second.AccessToken != "token-1" {
		t.Fatalf("unexpected token values: %+v %+v", first, second)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("expected one token request before refresh, got %d", got)
	}

	now = now.Add(95 * time.Second)
	if _, err := ts.Token(context.Background()); err != nil {
		t.Fatalf("refresh token failed: %v", err)
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

	ts := NewTokenSource(New("client", "secret", "tenant", "", CloudPublic), server.Client())
	ts.tokenURL = server.URL + "/tenant/oauth2/v2.0/token"
	ts.clock = func() time.Time { return time.Unix(1700000000, 0) }

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, err := ts.Token(context.Background())
			if err != nil {
				t.Errorf("Token returned error: %v", err)
				return
			}
			if token.AccessToken != "shared-token" {
				t.Errorf("unexpected token: %q", token.AccessToken)
			}
		}()
	}
	wg.Wait()

	if got := requests.Load(); got != 1 {
		t.Fatalf("expected exactly one token request, got %d", got)
	}
}

func TestTokenSourceErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_client","error_description":"bad secret"}`))
	}))
	defer server.Close()

	ts := NewTokenSource(New("client", "secret", "tenant", "", CloudPublic), server.Client())
	ts.tokenURL = server.URL + "/tenant/oauth2/v2.0/token"

	_, err := ts.Token(context.Background())
	if err == nil {
		t.Fatal("expected token error")
	}
	if got := err.Error(); got != "azure oauth2: invalid_client: bad secret" {
		t.Fatalf("unexpected error: %s", got)
	}
}
