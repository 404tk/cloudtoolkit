package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/internal/testutil"
)

func TestClientDoGETAndPOST(t *testing.T) {
	var tokenCalls int
	var sawPost bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			tokenCalls++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case "/get":
			if got := r.Header.Get("Authorization"); got != "Bearer token" {
				t.Fatalf("unexpected authorization: %s", got)
			}
			if got := r.URL.Query().Get("a"); got != "1" {
				t.Fatalf("unexpected query a: %s", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"value":"ok"}`))
		case "/post":
			sawPost = true
			if got := r.Header.Get("Content-Type"); got != "application/json; charset=UTF-8" {
				t.Fatalf("unexpected content type: %s", got)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := newTestClient(server, nil)

	var payload map[string]string
	if err := client.Do(context.Background(), Request{
		Method:     http.MethodGet,
		BaseURL:    server.URL,
		Path:       "/get",
		Query:      url.Values{"a": {"1"}},
		Idempotent: true,
	}, &payload); err != nil {
		t.Fatalf("GET Do() error = %v", err)
	}
	if payload["value"] != "ok" {
		t.Fatalf("unexpected payload: %+v", payload)
	}

	if err := client.Do(context.Background(), Request{
		Method:  http.MethodPost,
		BaseURL: server.URL,
		Path:    "/post",
		Body:    []byte(`{"name":"demo"}`),
	}, nil); err != nil {
		t.Fatalf("POST Do() error = %v", err)
	}
	if !sawPost {
		t.Fatal("expected POST handler to run")
	}
	if tokenCalls != 1 {
		t.Fatalf("unexpected token calls: %d", tokenCalls)
	}
}

func TestClientRetryAfter429(t *testing.T) {
	var attempts int
	var slept []time.Duration
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case "/retry":
			attempts++
			if attempts == 1 {
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":{"code":429,"message":"slow down","status":"RESOURCE_EXHAUSTED","errors":[{"reason":"rateLimitExceeded"}]}}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"value":"ok"}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := newTestClient(server, retryPolicy{
		baseDelay: 0,
		sleep: func(_ context.Context, d time.Duration) error {
			slept = append(slept, d)
			return nil
		},
		rand:  func() float64 { return 0 },
		clock: func() time.Time { return time.Unix(1700000000, 0) },
	})

	var payload map[string]string
	if err := client.Do(context.Background(), Request{
		Method:     http.MethodGet,
		BaseURL:    server.URL,
		Path:       "/retry",
		Idempotent: true,
	}, &payload); err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if len(slept) != 1 || slept[0] != time.Second {
		t.Fatalf("unexpected sleeps: %v", slept)
	}
}

func TestClientReturnsAPIErrorWithoutRetryOn401(t *testing.T) {
	var attempts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case "/unauthorized":
			attempts++
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":401,"message":"expired","status":"UNAUTHENTICATED","errors":[{"reason":"authError"}]}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := newTestClient(server, nil)
	err := client.Do(context.Background(), Request{
		Method:     http.MethodGet,
		BaseURL:    server.URL,
		Path:       "/unauthorized",
		Idempotent: true,
	}, nil)
	if err == nil {
		t.Fatal("expected APIError")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized || apiErr.Status != "UNAUTHENTICATED" {
		t.Fatalf("unexpected api error: %+v", apiErr)
	}
	if attempts != 1 {
		t.Fatalf("expected one attempt, got %d", attempts)
	}
}

func TestClientFallsBackForHTMLErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case "/html":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`<html>broken</html>`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := newTestClient(server, nil)
	err := client.Do(context.Background(), Request{
		Method:     http.MethodGet,
		BaseURL:    server.URL,
		Path:       "/html",
		Idempotent: true,
	}, nil)
	if err == nil || err.Error() != "gcp api error: status=500 body=<html>broken</html>" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func newTestClient(server *httptest.Server, policy RetryPolicy) *Client {
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
	opts := []Option{WithHTTPClient(httpClient)}
	if policy != nil {
		opts = append(opts, WithRetryPolicy(policy))
	}
	return NewClient(ts, opts...)
}
