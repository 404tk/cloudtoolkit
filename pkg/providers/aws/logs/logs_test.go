package logs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/aws/auth"
)

func newTestDriver(baseURL string) *Driver {
	return &Driver{
		Client: api.NewClient(
			auth.New("AKID", "SECRET", ""),
			api.WithBaseURL(baseURL),
			api.WithClock(func() time.Time { return time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC) }),
			api.WithRetryPolicy(api.RetryPolicy{
				MaxAttempts: 1,
				Sleep:       func(context.Context, time.Duration) error { return nil },
			}),
		),
		Region:        "us-east-1",
		DefaultRegion: "us-east-1",
	}
}

const sampleLogGroupsPage1 = `{"logGroups":[
  {"logGroupName":"/ctk/demo-app","creationTime":1713427200000,"retentionInDays":30,"storedBytes":1048576,"arn":"arn:aws:logs:us-east-1:123:log-group:/ctk/demo-app:*"},
  {"logGroupName":"/aws/lambda/healthcheck","creationTime":1713427260000,"storedBytes":4096,"arn":"arn:aws:logs:us-east-1:123:log-group:/aws/lambda/healthcheck:*"}
],"nextToken":"page-2"}`

const sampleLogGroupsPage2 = `{"logGroups":[
  {"logGroupName":"/aws/eks/audit","creationTime":1713427320000,"storedBytes":2048,"arn":"arn:aws:logs:us-east-1:123:log-group:/aws/eks/audit:*"}
]}`

func TestGetLogsParsesAndPaginates(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Amz-Target"); got != "Logs_20140328.DescribeLogGroups" {
			t.Errorf("unexpected target: %s", got)
		}
		calls++
		if calls == 1 {
			_, _ = w.Write([]byte(sampleLogGroupsPage1))
		} else {
			_, _ = w.Write([]byte(sampleLogGroupsPage2))
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	logs, err := driver.GetLogs(context.Background())
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 log groups, got %d", len(logs))
	}
	if logs[0].ProjectName != "/ctk/demo-app" {
		t.Errorf("unexpected first log group: %+v", logs[0])
	}
	if logs[0].Region != "us-east-1" {
		t.Errorf("expected us-east-1, got %s", logs[0].Region)
	}
	if logs[0].LastModifyTime == "" {
		t.Errorf("expected formatted creation time, got empty")
	}
	if !strings.Contains(logs[0].Description, "arn:aws:logs:us-east-1") {
		t.Errorf("expected ARN in description, got %q", logs[0].Description)
	}
	if calls != 2 {
		t.Errorf("expected 2 paginated calls, got %d", calls)
	}
}

func TestGetLogsPropagatesAccessDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		http.Error(w, `{"__type":"AccessDeniedException","Message":"User is not authorized"}`, http.StatusBadRequest)
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	// "all" mode probes the first region and surfaces AccessDenied as a fatal
	// error so the caller can short-circuit. Single-region calls keep the
	// failure on PartialError() so cloudlist can still surface other resources.
	driver.Region = "all"
	driver.AvailableRegions = []string{"us-east-1"}
	_, err := driver.GetLogs(context.Background())
	if err == nil {
		t.Fatal("expected error when DescribeLogGroups fails")
	}
	if !strings.Contains(err.Error(), "AccessDenied") {
		t.Errorf("expected AccessDenied propagation, got %v", err)
	}
}

func TestGetLogsKeepsSingleRegionFailureOnPartialError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"__type":"InternalServerException","Message":"transient"}`, http.StatusInternalServerError)
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	logs, err := driver.GetLogs(context.Background())
	if err != nil {
		t.Fatalf("GetLogs returned fatal err for single-region transient: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 logs on transient failure, got %d", len(logs))
	}
	if pe := driver.PartialError(); pe == nil {
		t.Fatal("expected PartialError to capture transient failure")
	}
}

func TestGetLogsReturnsEmptyOnNilGroupList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"logGroups":[]}`))
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	logs, err := driver.GetLogs(context.Background())
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 log groups, got %d", len(logs))
	}
}
