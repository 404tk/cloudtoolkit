package sms

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/auth"
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

func TestGetResourceListsSignsAndTemplates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/signs"):
			_, _ = w.Write([]byte(`{"requestId":"r1","result":{"signs":[
  {"signId":"sg-1","signName":"ctk-prod","signType":"app","status":"PASSED"},
  {"signId":"sg-2","signName":"ctk-stage","signType":"website","status":"PENDING"}
],"totalCount":2}}`))
		case strings.HasSuffix(r.URL.Path, "/templates"):
			_, _ = w.Write([]byte(`{"requestId":"r2","result":{"templates":[
  {"templateId":"tpl-1","templateName":"OTP","templateContent":"Code is {1}","status":"PASSED"}
],"totalCount":1}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-north-1"}
	res, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource: %v", err)
	}
	if len(res.Signs) != 2 || res.Signs[0].Name != "ctk-prod" || res.Signs[0].Status != "PASSED" {
		t.Errorf("signs mismatch: %+v", res.Signs)
	}
	if len(res.Templates) != 1 || res.Templates[0].Name != "OTP" {
		t.Errorf("templates mismatch: %+v", res.Templates)
	}
}

func TestGetResourcePropagatesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"requestId":"r-err","error":{"code":"AccessDenied","message":"forbidden"}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-north-1"}
	_, err := driver.GetResource(context.Background())
	if err == nil {
		t.Fatal("expected error when describeSigns fails")
	}
	if !strings.Contains(err.Error(), "AccessDenied") {
		t.Errorf("expected AccessDenied in err, got %v", err)
	}
}

func TestGetResourceHandlesEmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/signs"):
			_, _ = w.Write([]byte(`{"requestId":"r1","result":{"signs":[],"totalCount":0}}`))
		case strings.HasSuffix(r.URL.Path, "/templates"):
			_, _ = w.Write([]byte(`{"requestId":"r2","result":{"templates":[],"totalCount":0}}`))
		}
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-north-1"}
	res, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource: %v", err)
	}
	if len(res.Signs) != 0 || len(res.Templates) != 0 {
		t.Errorf("expected empty result, got %+v", res)
	}
}
