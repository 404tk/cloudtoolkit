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
		case "GetSubAccountList":
			if values.Get("Version") != "2021-01-11" {
				t.Fatalf("unexpected subaccount version: %s", values.Get("Version"))
			}
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"r0"},"Result":{"total":1,"list":[
  {"subAccountId":"sms-sub-1","subAccountName":"ctk-validation","status":1}
]}}`))
		case "GetSignatureAndOrderList":
			if values.Get("Version") != "2025-01-01" {
				t.Fatalf("unexpected sign version: %s", values.Get("Version"))
			}
			if values.Get("subAccount") != "sms-sub-1" {
				t.Fatalf("unexpected sign subAccount: %s", values.Get("subAccount"))
			}
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"r1"},"Result":{"total":2,"list":[
  {"id":"sg-1","content":"ctk-prod","source":"company","status":3},
  {"id":"sg-2","content":"ctk-stage","source":"website","status":1}
]}}`))
		case "GetSmsTemplateAndOrderList":
			if values.Get("Version") != "2021-01-11" {
				t.Fatalf("unexpected template version: %s", values.Get("Version"))
			}
			if values.Get("subAccount") != "sms-sub-1" {
				t.Fatalf("unexpected template subAccount: %s", values.Get("subAccount"))
			}
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"r2"},"Result":{"total":1,"list":[
  {"id":"tpl-1","name":"OTP","channelType":"verification","content":"Code is {1}","status":3}
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
	if len(res.Signs) != 2 || res.Signs[0].Name != "ctk-prod" || res.Signs[0].Status != "passed" {
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
		t.Fatal("expected error when signature list fails")
	}
	if !strings.Contains(err.Error(), "AccessDenied") {
		t.Errorf("expected AccessDenied, got %v", err)
	}
}

func TestGetResourceHandlesEmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"r3"},"Result":{"total":0,"list":[]}}`))
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
