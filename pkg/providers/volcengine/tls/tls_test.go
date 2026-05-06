package tls

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

func TestGetLogsListsTLSProjects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values, _ := url.ParseQuery(r.URL.RawQuery)
		if values.Get("Action") != "DescribeProjects" {
			t.Fatalf("unexpected action: %s", values.Get("Action"))
		}
		_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"r1"},"Result":{"Total":2,"Projects":[
  {"ProjectId":"tls-1","ProjectName":"prod-tls","Region":"cn-beijing","CreateTime":"2026-04-15 09:11:00","Description":"prod logs"},
  {"ProjectId":"tls-2","ProjectName":"audit-tls","Region":"cn-beijing","CreateTime":"2026-04-16 10:00:00","Description":""}
]}}`))
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	logs, err := driver.GetLogs(context.Background())
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(logs))
	}
	if logs[0].ProjectName != "prod-tls" || logs[0].Region != "cn-beijing" {
		t.Errorf("unexpected first row: %+v", logs[0])
	}
	if logs[0].Description != "prod logs" {
		t.Errorf("description should carry through, got %q", logs[0].Description)
	}
}

func TestGetLogsRejectsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"ResponseMetadata":{"Error":{"Code":"AccessDenied","Message":"forbidden"}}}`))
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	_, err := driver.GetLogs(context.Background())
	if err == nil {
		t.Fatal("expected error when DescribeProjects fails")
	}
	if !strings.Contains(err.Error(), "AccessDenied") {
		t.Errorf("expected AccessDenied in err, got %v", err)
	}
}

func TestGetLogsHandlesEmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"r2"},"Result":{"Total":0,"Projects":[]}}`))
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	logs, err := driver.GetLogs(context.Background())
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 projects, got %d", len(logs))
	}
}
