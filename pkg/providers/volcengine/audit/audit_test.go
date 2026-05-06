package audit

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

func newTestClient(baseURL string) *api.Client {
	return api.NewClient(
		auth.New("AKID", "SECRET", ""),
		api.WithBaseURL(baseURL),
		api.WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
		api.WithRetryPolicy(api.RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)
}

func TestDumpEventsParsesPaging(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		values, _ := url.ParseQuery(r.URL.RawQuery)
		if values.Get("Action") != "DescribeEvents" {
			t.Fatalf("unexpected action: %s", values.Get("Action"))
		}
		if calls == 1 {
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"r1"},"Result":{"Events":[{"EventId":"e1","EventName":"CreateUser","EventTime":"2026-04-22T09:11:00Z","SourceIPAddress":"1.1.1.1","Status":"Success","AccessKeyId":"AKLT"}],"PageToken":"p2"}}`))
			return
		}
		if got := values.Get("PageToken"); got != "p2" {
			t.Fatalf("expected page token p2 on second call, got %q", got)
		}
		_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"r2"},"Result":{"Events":[{"EventId":"e2","EventName":"DeleteUser","EventTime":"2026-04-22T09:14:00Z","Status":"Failed"}],"PageToken":""}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-beijing"}
	events, err := driver.DumpEvents(context.Background(), "")
	if err != nil {
		t.Fatalf("DumpEvents: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Id != "e1" || events[0].Name != "CreateUser" {
		t.Errorf("unexpected first event: %+v", events[0])
	}
	if events[1].Status != "Failed" {
		t.Errorf("unexpected second status: %+v", events[1])
	}
}

func TestDumpEventsParsesTimeWindow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values, _ := url.ParseQuery(r.URL.RawQuery)
		if values.Get("StartTime") != "1700000000" {
			t.Fatalf("expected start unix 1700000000, got %s", values.Get("StartTime"))
		}
		if values.Get("EndTime") != "1700003600" {
			t.Fatalf("expected end unix 1700003600, got %s", values.Get("EndTime"))
		}
		_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"r1"},"Result":{"Events":[],"PageToken":""}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-beijing"}
	if _, err := driver.DumpEvents(context.Background(), "1700000000:1700003600"); err != nil {
		t.Fatalf("DumpEvents: %v", err)
	}
}

func TestDumpEventsRejectsMalformedWindow(t *testing.T) {
	driver := &Driver{Client: newTestClient("http://example.invalid"), Region: "cn-beijing"}
	if _, err := driver.DumpEvents(context.Background(), "not-a-window"); err == nil {
		t.Fatalf("expected error for malformed window")
	}
}

func TestDumpEventsPropagatesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"r1","Error":{"Code":"Forbidden","Message":"audit not enabled"}}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-beijing"}
	_, err := driver.DumpEvents(context.Background(), "")
	if err == nil {
		t.Fatalf("expected error for 403")
	}
	if !strings.Contains(err.Error(), "Forbidden") {
		t.Errorf("expected Forbidden in error, got %v", err)
	}
}

func TestHandleEventsReturnsUnsupported(t *testing.T) {
	driver := &Driver{Client: newTestClient("http://example.invalid"), Region: "cn-beijing"}
	if _, err := driver.HandleEvents(context.Background(), ""); err == nil {
		t.Fatalf("expected error from HandleEvents")
	}
}
