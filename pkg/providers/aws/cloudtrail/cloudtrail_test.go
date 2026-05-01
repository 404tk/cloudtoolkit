package cloudtrail

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/aws/auth"
)

func newTestDriver(t *testing.T, baseURL string) *Driver {
	t.Helper()
	client := api.NewClient(
		auth.New("AKID", "SECRET", ""),
		api.WithBaseURL(baseURL),
		api.WithClock(func() time.Time { return time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC) }),
		api.WithRetryPolicy(api.RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)
	return &Driver{Client: client, Region: "us-east-1", DefaultRegion: "us-east-1"}
}

func TestDumpEventsParsesAndPaginates(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Amz-Target"); !strings.HasSuffix(got, ".LookupEvents") {
			t.Fatalf("unexpected target: %s", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/x-amz-json-1.1" {
			t.Fatalf("unexpected content-type: %s", got)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		calls++
		if calls == 1 {
			if _, ok := body["NextToken"]; ok {
				t.Fatalf("first call should not include NextToken, got: %v", body)
			}
			_, _ = w.Write([]byte(`{"NextToken":"page2","Events":[
				{"EventId":"e1","EventName":"CreateUser","EventTime":1714694400,"AccessKeyId":"AK1","Resources":[{"ResourceType":"AWS::IAM::User","ResourceName":"alice"}]},
				{"EventId":"e2","EventName":"AttachUserPolicy","EventTime":1714694430,"AccessKeyId":"AK1","Resources":[{"ResourceType":"AWS::IAM::Policy","ResourceName":"AdministratorAccess"}]}
			]}`))
			return
		}
		if got, _ := body["NextToken"].(string); got != "page2" {
			t.Fatalf("expected NextToken=page2, got %q", got)
		}
		_, _ = w.Write([]byte(`{"NextToken":"","Events":[
			{"EventId":"e3","EventName":"RunInstances","EventTime":1714694460,"AccessKeyId":"AK1","Resources":[{"ResourceType":"AWS::EC2::Instance","ResourceName":"i-0a"}]}
		]}`))
	}))
	defer server.Close()

	driver := newTestDriver(t, server.URL)
	events, err := driver.DumpEvents(context.Background(), "")
	if err != nil {
		t.Fatalf("DumpEvents: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 paginated calls, got %d", calls)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].Id != "e1" || events[0].Affected != "alice" || events[0].API != "CreateUser" {
		t.Fatalf("unexpected first event: %+v", events[0])
	}
	if events[2].Time == "" {
		t.Fatalf("expected formatted time on event[2], got empty")
	}
}

func TestDumpEventsTimeWindowSent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			StartTime *float64 `json:"StartTime"`
			EndTime   *float64 `json:"EndTime"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.StartTime == nil || *body.StartTime != 1714000000 {
			t.Fatalf("expected StartTime=1714000000, got %+v", body.StartTime)
		}
		if body.EndTime == nil || *body.EndTime != 1714003600 {
			t.Fatalf("expected EndTime=1714003600, got %+v", body.EndTime)
		}
		_, _ = w.Write([]byte(`{"NextToken":"","Events":[]}`))
	}))
	defer server.Close()

	driver := newTestDriver(t, server.URL)
	if _, err := driver.DumpEvents(context.Background(), "1714000000:1714003600"); err != nil {
		t.Fatalf("DumpEvents: %v", err)
	}
}

func TestDumpEventsRejectsBadWindow(t *testing.T) {
	driver := newTestDriver(t, "http://example.invalid")
	if _, err := driver.DumpEvents(context.Background(), "abc:def"); err == nil {
		t.Fatalf("expected parse error, got nil")
	}
	if _, err := driver.DumpEvents(context.Background(), "200:100"); err == nil {
		t.Fatalf("expected end<start error, got nil")
	}
}

func TestDumpEventsServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"__type":"AccessDenied","Message":"not allowed"}`))
	}))
	defer server.Close()

	driver := newTestDriver(t, server.URL)
	if _, err := driver.DumpEvents(context.Background(), ""); err == nil {
		t.Fatalf("expected error from server, got nil")
	}
}

func TestHandleEventsUnsupported(t *testing.T) {
	driver := newTestDriver(t, "http://example.invalid")
	_, err := driver.HandleEvents(context.Background(), "")
	if err == nil {
		t.Fatalf("expected unsupported error, got nil")
	}
	if !strings.Contains(err.Error(), "whitelist") {
		t.Fatalf("expected whitelist mention in error, got %q", err.Error())
	}
}
