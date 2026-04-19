package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
)

func TestClientDoJSONGETWithQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/users" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.RawQuery; got != "limit=10&marker=abc" {
			t.Fatalf("unexpected query: %s", got)
		}
		if got := r.Header.Get(HeaderAuthorization); got == "" {
			t.Fatal("missing authorization header")
		}
		_, _ = w.Write([]byte(`{"users":[]}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	var out ListUsersResponse
	err := client.DoJSON(context.Background(), Request{
		Service: "iam",
		Region:  "cn-north-4",
		Method:  http.MethodGet,
		Path:    "/v3/users",
		Query: url.Values{
			"limit":  {"10"},
			"marker": {"abc"},
		},
	}, &out)
	if err != nil {
		t.Fatalf("DoJSON() error = %v", err)
	}
}

func TestClientDoJSONPostSetsContentHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/json;charset=UTF-8" {
			t.Fatalf("unexpected content type: %s", got)
		}
		if got := r.Header.Get(HeaderContentSha256); got != "" {
			t.Fatalf("unexpected content hash header: %s", got)
		}
		if got := r.Header.Get(HeaderAuthorization); !strings.Contains(got, "SignedHeaders=x-sdk-date") {
			t.Fatalf("unexpected authorization header: %s", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if string(body) != `{"user":{"name":"ctk"}}` {
			t.Fatalf("unexpected body: %s", string(body))
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	err := client.DoJSON(context.Background(), Request{
		Service: "iam",
		Region:  "cn-north-4",
		Method:  http.MethodPost,
		Path:    "/v3/users",
		Body:    []byte(`{"user":{"name":"ctk"}}`),
	}, &struct {
		OK bool `json:"ok"`
	}{})
	if err != nil {
		t.Fatalf("DoJSON() error = %v", err)
	}
}

func TestClientDoJSONAllowsEmptyBodyResponses(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "200-empty", statusCode: 200},
		{name: "204-empty", statusCode: 204},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			client := newTestClient(server.URL)
			if err := client.DoJSON(context.Background(), Request{
				Service: "iam",
				Region:  "cn-north-4",
				Method:  http.MethodDelete,
				Path:    "/v3/users/u-1",
			}, nil); err != nil {
				t.Fatalf("DoJSON() error = %v", err)
			}
		})
	}
}

func TestClientDoJSONReturnsAPIError(t *testing.T) {
	tests := []struct {
		name string
		body string
		code string
		msg  string
	}{
		{
			name: "legacy",
			body: `{"error_code":"IAM.0001","error_msg":"bad request"}`,
			code: "IAM.0001",
			msg:  "bad request",
		},
		{
			name: "keystone",
			body: `{"error":{"code":"IAM.0002","message":"forbidden"}}`,
			code: "IAM.0002",
			msg:  "forbidden",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Request-Id", "req-1")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			client := newTestClient(server.URL)
			err := client.DoJSON(context.Background(), Request{
				Service: "iam",
				Region:  "cn-north-4",
				Method:  http.MethodGet,
				Path:    "/v3/users",
			}, nil)
			apiErr, ok := err.(*APIError)
			if !ok {
				t.Fatalf("expected *APIError, got %T", err)
			}
			if apiErr.Code != tc.code || apiErr.Message != tc.msg || apiErr.RequestID != "req-1" {
				t.Fatalf("unexpected api error: %+v", apiErr)
			}
		})
	}
}

func TestClientDoJSONReturnsFallbackError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	err := client.DoJSON(context.Background(), Request{
		Service: "iam",
		Region:  "cn-north-4",
		Method:  http.MethodGet,
		Path:    "/v3/users",
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "status=400") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientDoJSONRetriesIdempotentGET(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error_code":"IAM.0502","error_msg":"bad gateway"}`))
			return
		}
		_, _ = w.Write([]byte(`{"users":[]}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	var out ListUsersResponse
	if err := client.DoJSON(context.Background(), Request{
		Service: "iam",
		Region:  "cn-north-4",
		Method:  http.MethodGet,
		Path:    "/v3/users",
	}, &out); err != nil {
		t.Fatalf("DoJSON() error = %v", err)
	}
	if attempts != 2 {
		t.Fatalf("unexpected attempts: %d", attempts)
	}
}

func TestClientDoJSONDoesNotRetryPost(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error_code":"IAM.0502","error_msg":"bad gateway"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	err := client.DoJSON(context.Background(), Request{
		Service: "iam",
		Region:  "cn-north-4",
		Method:  http.MethodPost,
		Path:    "/v3/users",
		Body:    []byte(`{"user":{"name":"ctk"}}`),
	}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Fatalf("unexpected attempts: %d", attempts)
	}
}

func TestClientDoJSONRetriesNetworkErrorForGET(t *testing.T) {
	attempts := 0
	client := NewClient(
		auth.New("AKID", "SECRET", "cn-north-4", false),
		WithHTTPClient(&http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				attempts++
				if attempts == 1 {
					return nil, errors.New("boom")
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"users":[]}`)),
					Header:     http.Header{},
					Request:    r,
				}, nil
			}),
		}),
		WithRetryPolicy(retryPolicy{
			baseDelay: 0,
			sleep:     func(context.Context, time.Duration) error { return nil },
			rand:      func() float64 { return 0 },
		}),
		WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
	)

	var out ListUsersResponse
	if err := client.DoJSON(context.Background(), Request{
		Service: "iam",
		Region:  "cn-north-4",
		Method:  http.MethodGet,
		Path:    "/v3/users",
	}, &out); err != nil {
		t.Fatalf("DoJSON() error = %v", err)
	}
	if attempts != 2 {
		t.Fatalf("unexpected attempts: %d", attempts)
	}
}

func newTestClient(baseURL string) *Client {
	return NewClient(
		auth.New("AKID", "SECRET", "cn-north-4", false),
		WithBaseURL(baseURL),
		WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
		WithRetryPolicy(retryPolicy{
			baseDelay: 0,
			sleep:     func(context.Context, time.Duration) error { return nil },
			rand:      func() float64 { return 0 },
		}),
	)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}
