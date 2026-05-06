package sms

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/auth"
)

func newTestDriver(baseURL string) *Driver {
	client := api.NewClient(
		auth.New("AKID", "SECRET", ""),
		api.WithBaseURL(baseURL),
		api.WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
		api.WithRetryPolicy(api.RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)
	return &Driver{Client: client, Region: "cn-beijing"}
}

func TestGetResourceListsSignsAndTemplates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values, _ := url.ParseQuery(r.URL.RawQuery)
		switch values.Get("Action") {
		case "ListSign":
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"r1"},"Result":{"Total":2,"List":[
  {"SignId":"sg-1","Sign":"ctk-prod","SignType":"company","Status":"PASSED"},
  {"SignId":"sg-2","Sign":"ctk-stage","SignType":"website","Status":"PENDING"}
]}}`))
		case "ListSmsTemplate":
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"r2"},"Result":{"Total":1,"List":[
  {"TemplateId":"tpl-1","TemplateName":"OTP","TemplateType":"verification","Content":"Code is {1}","Status":"PASSED"}
]}}`))
		default:
			t.Fatalf("unexpected action: %s", values.Get("Action"))
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
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
		_, _ = w.Write([]byte(`{"ResponseMetadata":{"Error":{"Code":"AccessDenied","Message":"forbidden"}}}`))
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	_, err := driver.GetResource(context.Background())
	if err == nil {
		t.Fatal("expected error when ListSign fails")
	}
	if !strings.Contains(err.Error(), "AccessDenied") {
		t.Errorf("expected AccessDenied, got %v", err)
	}
}

func TestGetResourceHandlesEmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"r3"},"Result":{"Total":0,"List":[]}}`))
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	res, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource: %v", err)
	}
	if len(res.Signs) != 0 || len(res.Templates) != 0 {
		t.Errorf("expected empty result, got %+v", res)
	}
}
