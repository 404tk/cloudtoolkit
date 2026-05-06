package cls

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
)

func newTestDriver(t *testing.T, baseURL string) *Driver {
	t.Helper()
	d := &Driver{Credential: auth.New("ak", "sk", ""), Region: "ap-guangzhou"}
	d.SetClientOptions(
		api.WithBaseURL(baseURL),
		api.WithClock(func() time.Time { return time.Unix(1776458501, 0).UTC() }),
		api.WithRetryPolicy(api.RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)
	return d
}

func TestGetLogsMapsLogsets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-TC-Action"); got != "DescribeLogsets" {
			t.Fatalf("unexpected action: %s", got)
		}
		_, _ = w.Write([]byte(`{"Response":{"TotalCount":2,"Logsets":[
  {"LogsetId":"ls-1","LogsetName":"prod","CreateTime":"2026-04-15 09:11:00","TopicCount":3},
  {"LogsetId":"ls-2","LogsetName":"audit","CreateTime":"2026-04-16 10:00:00","TopicCount":1}
],"RequestId":"r1"}}`))
	}))
	defer server.Close()

	driver := newTestDriver(t, server.URL)
	logs, err := driver.GetLogs(context.Background())
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 logsets, got %d", len(logs))
	}
	if logs[0].ProjectName != "prod" || logs[0].Region != "ap-guangzhou" {
		t.Errorf("unexpected logset: %+v", logs[0])
	}
	if !strings.Contains(logs[0].Description, "logsetId=ls-1") {
		t.Errorf("expected logsetId in description, got %q", logs[0].Description)
	}
	if logs[1].LastModifyTime != "2026-04-16 10:00:00" {
		t.Errorf("unexpected create time: %q", logs[1].LastModifyTime)
	}
}

func TestGetLogsRejectsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Response":{"Error":{"Code":"AuthFailure.SignatureFailure","Message":"signature mismatch"},"RequestId":"r-err"}}`))
	}))
	defer server.Close()

	driver := newTestDriver(t, server.URL)
	_, err := driver.GetLogs(context.Background())
	if err == nil {
		t.Fatal("expected error when DescribeLogsets returns Error")
	}
	if !strings.Contains(err.Error(), "AuthFailure") {
		t.Errorf("expected AuthFailure in err, got %v", err)
	}
}

func TestGetLogsHandlesEmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Response":{"TotalCount":0,"Logsets":[],"RequestId":"r2"}}`))
	}))
	defer server.Close()

	driver := newTestDriver(t, server.URL)
	logs, err := driver.GetLogs(context.Background())
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 logsets, got %d", len(logs))
	}
}
