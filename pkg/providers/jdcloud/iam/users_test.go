package iam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/auth"
)

func TestDriverListUsersMapsUsers(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/subUsers" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		requests++
		switch requests {
		case 1:
			if got := r.URL.Query().Get("pageNumber"); got != "1" {
				t.Fatalf("unexpected pageNumber: %s", got)
			}
			if got := r.URL.Query().Get("pageSize"); got != "100" {
				t.Fatalf("unexpected pageSize: %s", got)
			}
			_, _ = w.Write([]byte(`{"requestId":"req-1","result":{"subUsers":[{"name":"alice","account":"1001","createTime":"2026-04-19T12:00:00Z"}],"total":2}}`))
		case 2:
			if got := r.URL.Query().Get("pageNumber"); got != "2" {
				t.Fatalf("unexpected pageNumber: %s", got)
			}
			if got := r.URL.Query().Get("pageSize"); got != "100" {
				t.Fatalf("unexpected pageSize: %s", got)
			}
			_, _ = w.Write([]byte(`{"requestId":"req-2","result":{"subUsers":[{"name":"bob","account":"1002","createTime":"2026-04-19T13:00:00Z"}],"total":2}}`))
		default:
			t.Fatalf("unexpected request count: %d", requests)
		}
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL)}
	got, err := driver.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(got) != 2 || got[0].UserName != "alice" || got[1].UserId != "1002" {
		t.Fatalf("unexpected users: %+v", got)
	}
	if requests != 2 {
		t.Fatalf("unexpected request count: %d", requests)
	}
}

func TestDriverValidatorTreats404AsValid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/subUsers" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("pageNumber"); got != "1" {
			t.Fatalf("unexpected pageNumber: %s", got)
		}
		if got := r.URL.Query().Get("pageSize"); got != "1" {
			t.Fatalf("unexpected pageSize: %s", got)
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"requestId":"req-404","error":{"status":"NOT_FOUND","code":404,"message":"not found"}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL)}
	if !driver.Validator("missing-user") {
		t.Fatal("expected validator to accept 404 as reachable credential")
	}
}

func TestDriverValidatorTreats403AsValid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/subUsers" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"requestId":"req-403","error":{"status":"HTTP_FORBIDDEN","code":403,"message":"this api is not allowed [gw]"}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL)}
	if !driver.Validator("forbidden-user") {
		t.Fatal("expected validator to accept inconclusive 403 gateway response")
	}
}

func TestDriverValidatorRejects401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/subUsers" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"requestId":"req-401","error":{"status":"HTTP_UNAUTHORIZED","code":401,"message":"unauthorized"}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL)}
	if driver.Validator("unauthorized-user") {
		t.Fatal("expected validator to reject 401 auth failure")
	}
}

func newTestClient(baseURL string) *api.Client {
	return api.NewClient(
		auth.New("AKID", "SECRET", ""),
		api.WithBaseURL(baseURL),
		api.WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
		api.WithNonceFunc(func() string { return "ebf8b26d-c3be-402f-9f10-f8b6573fd823" }),
		api.WithRetryPolicy(api.RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)
}
