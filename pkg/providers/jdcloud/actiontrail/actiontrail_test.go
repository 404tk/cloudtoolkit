package actiontrail

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

func TestDumpEventsParses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/events:lookup") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"requestId":"r1","result":{"events":[{"eventId":"e1","eventName":"CreateSubUser","eventTime":"2026-04-22T09:11:00Z","sourceIpAddress":"203.0.113.62","status":"Success","accessKeyId":"JDC_X"},{"eventId":"e2","eventName":"DeleteAccessKey","eventTime":"2026-04-22T09:14:00Z","status":"Failed"}]}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-north-1"}
	events, err := driver.DumpEvents(context.Background(), "")
	if err != nil {
		t.Fatalf("DumpEvents: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Id != "e1" || events[0].Name != "CreateSubUser" {
		t.Errorf("unexpected first event: %+v", events[0])
	}
	if events[1].Status != "Failed" {
		t.Errorf("unexpected second status: %+v", events[1])
	}
}

func TestDumpEventsParsesTimeWindow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("startTime") != "1700000000" || query.Get("endTime") != "1700003600" {
			t.Fatalf("unexpected window: %s/%s", query.Get("startTime"), query.Get("endTime"))
		}
		_, _ = w.Write([]byte(`{"requestId":"r1","result":{"events":[]}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-north-1"}
	if _, err := driver.DumpEvents(context.Background(), "1700000000:1700003600"); err != nil {
		t.Fatalf("DumpEvents: %v", err)
	}
}

func TestDumpEventsRejectsMalformedWindow(t *testing.T) {
	driver := &Driver{Client: newTestClient("http://example.invalid"), Region: "cn-north-1"}
	if _, err := driver.DumpEvents(context.Background(), "garbage"); err == nil {
		t.Fatalf("expected error for malformed window")
	}
}

func TestDumpEventsPropagatesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"requestId":"r1","error":{"code":403,"message":"audit not enabled"}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-north-1"}
	_, err := driver.DumpEvents(context.Background(), "")
	if err == nil {
		t.Fatalf("expected error for 403")
	}
	if !strings.Contains(err.Error(), "audit not enabled") {
		t.Errorf("expected error to contain message, got %v", err)
	}
}

func TestHandleEventsReturnsUnsupported(t *testing.T) {
	driver := &Driver{Client: newTestClient("http://example.invalid"), Region: "cn-north-1"}
	if _, err := driver.HandleEvents(context.Background(), ""); err == nil {
		t.Fatalf("expected error from HandleEvents")
	}
}
