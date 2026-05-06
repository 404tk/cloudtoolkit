package logs

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

func TestGetLogsListsLogTopics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/logTopics:describe") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"requestId":"r1","result":{"logTopics":[
  {"logTopicId":"lt-1","logTopicName":"prod-app","logSetId":"ls-1","logSetName":"prod","createTime":"2026-04-15T09:11:00Z","description":"prod app logs"},
  {"logTopicId":"lt-2","logTopicName":"audit","logSetId":"ls-1","logSetName":"prod","createTime":"2026-04-16T10:00:00Z"}
],"totalCount":2}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-north-1"}
	logs, err := driver.GetLogs(context.Background())
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 topics, got %d", len(logs))
	}
	if logs[0].ProjectName != "prod/prod-app" {
		t.Errorf("expected logset/topic in name, got %q", logs[0].ProjectName)
	}
	if logs[0].Region != "cn-north-1" {
		t.Errorf("unexpected region: %s", logs[0].Region)
	}
	if logs[0].Description != "prod app logs" {
		t.Errorf("expected description carry-through, got %q", logs[0].Description)
	}
}

func TestGetLogsRejectsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"requestId":"r-err","error":{"code":"AccessDenied","message":"forbidden"}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-north-1"}
	_, err := driver.GetLogs(context.Background())
	if err == nil {
		t.Fatal("expected error when describeLogTopics fails")
	}
	if !strings.Contains(err.Error(), "AccessDenied") {
		t.Errorf("expected AccessDenied in err, got %v", err)
	}
}

func TestGetLogsHandlesEmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"requestId":"r2","result":{"logTopics":[],"totalCount":0}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-north-1"}
	logs, err := driver.GetLogs(context.Background())
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 topics, got %d", len(logs))
	}
}
