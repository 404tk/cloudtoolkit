package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/azure/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/cloud"
)

func TestClientDoGETAndPOST(t *testing.T) {
	var tokenCalls int
	var sawPost bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/tenant/oauth2/v2.0/token":
			tokenCalls++
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), "scope=https%3A%2F%2Fmanagement.azure.com%2F.default") {
				t.Fatalf("unexpected token scope body: %s", string(body))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case r.URL.Path == "/subscriptions" && r.Method == http.MethodGet:
			if got := r.Header.Get("Authorization"); got != "Bearer token" {
				t.Fatalf("unexpected auth header: %s", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"value":[{"subscriptionId":"sub-1","displayName":"one"}]}`))
		case r.URL.Path == "/post" && r.Method == http.MethodPost:
			sawPost = true
			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("unexpected content type: %s", got)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client := newTestClient(t, server)

	var subs ListSubscriptionsResponse
	if err := client.Do(context.Background(), Request{
		Method:     http.MethodGet,
		Path:       "/subscriptions",
		Query:      url.Values{"api-version": {SubscriptionsAPIVersion}},
		Idempotent: true,
	}, &subs); err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	if len(subs.Value) != 1 || subs.Value[0].SubscriptionID != "sub-1" {
		t.Fatalf("unexpected subscriptions: %+v", subs)
	}
	if err := client.Do(context.Background(), Request{
		Method:  http.MethodPost,
		Path:    "/post",
		Body:    []byte(`{"name":"demo"}`),
		Query:   url.Values{"api-version": {"2022-01-01"}},
		Headers: http.Header{"X-Test": {"1"}},
	}, nil); err != nil {
		t.Fatalf("POST failed: %v", err)
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
		case "/tenant/oauth2/v2.0/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case "/retry":
			attempts++
			if attempts == 1 {
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":{"code":"TooManyRequests","message":"slow down"}}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"value":[]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	httpClient := server.Client()
	httpClient.Transport = rewriteTokenHostTransport(t, httpClient.Transport, server.URL)
	ts := auth.NewTokenSource(auth.New("client", "secret", "tenant", "", auth.CloudPublic), httpClient)
	client := NewClient(
		ts,
		cloud.For(auth.CloudPublic),
		WithHTTPClient(httpClient),
		WithBaseURL(server.URL),
		WithRetryPolicy(retryPolicy{
			baseDelay: 0,
			sleep: func(_ context.Context, d time.Duration) error {
				slept = append(slept, d)
				return nil
			},
			rand: func() float64 { return 0 },
		}),
	)

	var payload map[string]any
	if err := client.Do(context.Background(), Request{
		Method:     http.MethodGet,
		Path:       "/retry",
		Query:      url.Values{"api-version": {"2021-01-01"}},
		Idempotent: true,
	}, &payload); err != nil {
		t.Fatalf("retry request failed: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if len(slept) != 1 || slept[0] != time.Second {
		t.Fatalf("unexpected sleep values: %v", slept)
	}
}

func TestClientReturnsAPIErrorWithoutRetryOn401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tenant/oauth2/v2.0/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600,"token_type":"Bearer"}`))
		case "/unauthorized":
			w.Header().Set("x-ms-request-id", "req-401")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":"Unauthorized","message":"expired"}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server)

	err := client.Do(context.Background(), Request{
		Method:     http.MethodGet,
		Path:       "/unauthorized",
		Query:      url.Values{"api-version": {"2021-01-01"}},
		Idempotent: true,
	}, nil)
	if err == nil {
		t.Fatal("expected API error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.RequestID != "req-401" || apiErr.Code != "Unauthorized" {
		t.Fatalf("unexpected api error: %+v", apiErr)
	}
}

func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	httpClient := server.Client()
	httpClient.Transport = rewriteTokenHostTransport(t, httpClient.Transport, server.URL)
	ts := auth.NewTokenSource(auth.New("client", "secret", "tenant", "", auth.CloudPublic), httpClient)
	return NewClient(ts, cloud.For(auth.CloudPublic), WithHTTPClient(httpClient), WithBaseURL(server.URL))
}

type tokenRewriteTransport struct {
	t      *testing.T
	base   http.RoundTripper
	target *url.URL
}

func rewriteTokenHostTransport(t *testing.T, base http.RoundTripper, rawTarget string) http.RoundTripper {
	t.Helper()
	if base == nil {
		base = http.DefaultTransport
	}
	target, err := url.Parse(rawTarget)
	if err != nil {
		t.Fatalf("parse target url: %v", err)
	}
	return tokenRewriteTransport{t: t, base: base, target: target}
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
