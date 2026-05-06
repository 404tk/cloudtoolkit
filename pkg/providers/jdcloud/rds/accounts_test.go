package rds

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/auth"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
)

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

func TestCreateAccountSendsExpectedPayload(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{RDSAccount: "ctkuser:Ctk!Pwd2026"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/instances/db-1/accounts") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"requestId":"r1"}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-north-1"}
	res, err := driver.CreateAccount(context.Background(), "db-1")
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if res.Username != "ctkuser" || res.Password != "Ctk!Pwd2026" {
		t.Errorf("unexpected result: %+v", res)
	}
}

func TestDeleteAccountSendsExpectedPayload(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{RDSAccount: "ctkuser:Ctk!Pwd2026"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/instances/db-1/accounts/ctkuser") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"requestId":"r1"}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-north-1"}
	res, err := driver.DeleteAccount(context.Background(), "db-1")
	if err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	if res.Username != "ctkuser" {
		t.Errorf("unexpected username: %s", res.Username)
	}
}

func TestCreateAccountRejectsMissingConfig(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{})
	driver := &Driver{Client: newTestClient("http://example.invalid"), Region: "cn-north-1"}
	if _, err := driver.CreateAccount(context.Background(), "db-1"); err == nil {
		t.Fatalf("expected error for missing rds-account-check")
	}
}

func TestDeleteAccountPropagatesAPIError(t *testing.T) {
	env.SetActiveForTest(t, &env.Env{RDSAccount: "ctkuser:Ctk!Pwd2026"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"requestId":"r1","error":{"code":1011,"message":"account not found"}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-north-1"}
	if _, err := driver.DeleteAccount(context.Background(), "db-1"); err == nil {
		t.Fatalf("expected error from DeleteAccount")
	} else if !strings.Contains(err.Error(), "account not found") {
		t.Errorf("expected error to mention account not found, got %v", err)
	}
}
