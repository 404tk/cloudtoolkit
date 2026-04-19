package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/auth"
)

func TestClientDoJSONBuildsSignedGETRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/v1/subUsers" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Host; got != "iam.jdcloud-api.com" {
			t.Fatalf("unexpected host: %s", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("unexpected content type: %s", got)
		}
		if got := r.Header.Get(HeaderXJdcloudDate); got != "20260419T120000Z" {
			t.Fatalf("unexpected x-jdcloud-date: %s", got)
		}
		if got := r.Header.Get(HeaderXJdcloudNonce); got != "ebf8b26d-c3be-402f-9f10-f8b6573fd823" {
			t.Fatalf("unexpected nonce: %s", got)
		}
		if got := r.Header.Get(HeaderXJdcloudToken); got != "token64" {
			t.Fatalf("unexpected token: %s", got)
		}
		authz := r.Header.Get(HeaderAuthorization)
		if !strings.Contains(authz, "/20260419/jdcloud-api/iam/jdcloud2_request") {
			t.Fatalf("unexpected authorization header: %s", authz)
		}
		_, _ = w.Write([]byte(`{"requestId":"req-1","result":{"subUsers":[{"name":"alice","account":"1001","createTime":"2026-04-19T12:00:00Z"}],"total":1}}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "token64", func() string {
		return "ebf8b26d-c3be-402f-9f10-f8b6573fd823"
	})

	var resp DescribeSubUsersResponse
	err := client.DoJSON(context.Background(), Request{
		Service: "iam",
		Region:  "",
		Method:  http.MethodGet,
		Version: "v1",
		Path:    "/subUsers",
	}, &resp)
	if err != nil {
		t.Fatalf("DoJSON() error = %v", err)
	}
	if len(resp.Result.SubUsers) != 1 || resp.Result.SubUsers[0].Name != "alice" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestClientDoJSONGeneratesUUIDv4Nonce(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(r.Header.Get(HeaderXJdcloudNonce)) {
			t.Fatalf("unexpected nonce: %s", r.Header.Get(HeaderXJdcloudNonce))
		}
		_, _ = w.Write([]byte(`{"requestId":"req-2","result":{"buckets":[]}}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "", nil)
	var resp ListBucketsResponse
	err := client.DoJSON(context.Background(), Request{
		Service: "oss",
		Region:  "cn-north-1",
		Method:  http.MethodGet,
		Version: "v1",
		Path:    "/regions/cn-north-1/buckets",
	}, &resp)
	if err != nil {
		t.Fatalf("DoJSON() error = %v", err)
	}
}

func TestClientDoJSONReturnsAPIErrorFromBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"requestId":"req-error","error":{"status":403,"code":40301,"message":"denied"}}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "", nil)
	err := client.DoJSON(context.Background(), Request{
		Service: "iam",
		Region:  "",
		Method:  http.MethodGet,
		Version: "v1",
		Path:    "/subUsers",
	}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Code != 40301 || apiErr.RequestID != "req-error" {
		t.Fatalf("unexpected api error: %+v", apiErr)
	}
}

func TestClientDoJSONHonorsCanceledContext(t *testing.T) {
	client := newTestClient("http://127.0.0.1:1", "", func() string {
		return "ebf8b26d-c3be-402f-9f10-f8b6573fd823"
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.DoJSON(ctx, Request{
		Service: "iam",
		Region:  "",
		Method:  http.MethodGet,
		Version: "v1",
		Path:    "/subUsers",
	}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func newTestClient(baseURL, token string, nonce func() string) *Client {
	opts := []Option{
		WithBaseURL(baseURL),
		WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
		WithRetryPolicy(RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	}
	if nonce != nil {
		opts = append(opts, WithNonceFunc(nonce))
	}
	return NewClient(auth.New("AKID", "SECRET", token), opts...)
}
