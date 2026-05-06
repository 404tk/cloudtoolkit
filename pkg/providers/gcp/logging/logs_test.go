package logging

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const sampleLogsListPage1 = `{"logNames":[
  "projects/proj-1/logs/cloudaudit.googleapis.com%2Factivity",
  "projects/proj-1/logs/cloudaudit.googleapis.com%2Fdata_access",
  "projects/proj-1/logs/compute.googleapis.com%2Fvpc_flows"
],"nextPageToken":"page-2"}`

const sampleLogsListPage2 = `{"logNames":[
  "projects/proj-1/logs/cloudaudit.googleapis.com%2Fsystem_event"
]}`

func TestGetLogsListsAndPaginates(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_, _ = w.Write([]byte(`{"access_token":"demo","token_type":"Bearer","expires_in":3600}`))
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/v2/projects/proj-1/logs") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		calls++
		if calls == 1 {
			_, _ = w.Write([]byte(sampleLogsListPage1))
		} else {
			_, _ = w.Write([]byte(sampleLogsListPage2))
		}
	}))
	defer server.Close()

	driver := &Driver{Client: newLoggingClient(t, server), Projects: []string{"proj-1"}}
	logs, err := driver.GetLogs(context.Background())
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}
	if len(logs) != 4 {
		t.Fatalf("expected 4 log names, got %d (%+v)", len(logs), logs)
	}
	if logs[0].ProjectName != "cloudaudit.googleapis.com/activity" {
		t.Errorf("expected URL-decoded short name, got %q", logs[0].ProjectName)
	}
	if logs[0].Region != "proj-1" {
		t.Errorf("expected project in Region column, got %q", logs[0].Region)
	}
	if !strings.HasPrefix(logs[0].Description, "projects/proj-1/logs/") {
		t.Errorf("expected full log path in description, got %q", logs[0].Description)
	}
}

func TestGetLogsRejectsListFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_, _ = w.Write([]byte(`{"access_token":"demo","token_type":"Bearer","expires_in":3600}`))
			return
		}
		http.Error(w, `{"error":{"code":403,"message":"forbidden","status":"PERMISSION_DENIED"}}`, http.StatusForbidden)
	}))
	defer server.Close()

	driver := &Driver{Client: newLoggingClient(t, server), Projects: []string{"proj-1"}}
	_, err := driver.GetLogs(context.Background())
	if err == nil {
		t.Fatal("expected error when projects/logs.list fails")
	}
	if !strings.Contains(err.Error(), "PERMISSION_DENIED") {
		t.Errorf("expected PERMISSION_DENIED in err, got %v", err)
	}
}

func TestGetLogsHandlesEmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_, _ = w.Write([]byte(`{"access_token":"demo","token_type":"Bearer","expires_in":3600}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newLoggingClient(t, server), Projects: []string{"proj-1"}}
	logs, err := driver.GetLogs(context.Background())
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 logs, got %d", len(logs))
	}
}
