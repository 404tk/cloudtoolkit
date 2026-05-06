package uact

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/auth"
)

func newDriver(baseURL string) *Driver {
	credential := auth.New("ucloudpubkey-EXAMPLE", "ucloudprivkey-EXAMPLE", "")
	return &Driver{
		Credential: credential,
		Client: api.NewClient(credential,
			api.WithBaseURL(baseURL),
			api.WithRetryPolicy(api.RetryPolicy{MaxAttempts: 1}),
		),
	}
}

func TestDumpEventsParses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if got := r.Form.Get("Action"); got != "DescribeActionLogList" {
			t.Fatalf("unexpected action: %s", got)
		}
		_, _ = w.Write([]byte(`{"Action":"DescribeActionLogListResponse","RetCode":0,"TotalCount":2,"Events":[{"EventId":"e1","EventName":"CreateUser","EventTime":"2026-04-22T09:11:00Z","SourceIPAddress":"1.1.1.1","Status":"Success","AccessKeyId":"UC1"},{"EventId":"e2","EventName":"DeleteAccessKey","EventTime":"2026-04-22T09:14:00Z","Status":"Failed"}]}`))
	}))
	defer server.Close()

	driver := newDriver(server.URL)
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
}

func TestDumpEventsParsesTimeWindow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if got := r.Form.Get("StartTime"); got != "1700000000" {
			t.Fatalf("expected start unix 1700000000, got %s", got)
		}
		if got := r.Form.Get("EndTime"); got != "1700003600" {
			t.Fatalf("expected end unix 1700003600, got %s", got)
		}
		_, _ = w.Write([]byte(`{"Action":"DescribeActionLogListResponse","RetCode":0,"TotalCount":0,"Events":[]}`))
	}))
	defer server.Close()

	driver := newDriver(server.URL)
	if _, err := driver.DumpEvents(context.Background(), "1700000000:1700003600"); err != nil {
		t.Fatalf("DumpEvents: %v", err)
	}
}

func TestDumpEventsRejectsMalformedWindow(t *testing.T) {
	driver := newDriver("http://example.invalid")
	if _, err := driver.DumpEvents(context.Background(), "garbage"); err == nil {
		t.Fatalf("expected error for malformed window")
	}
}

func TestDumpEventsPropagatesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Action":"DescribeActionLogListResponse","RetCode":230,"Message":"action trail not enabled"}`))
	}))
	defer server.Close()

	driver := newDriver(server.URL)
	_, err := driver.DumpEvents(context.Background(), "")
	if err == nil {
		t.Fatalf("expected error from DumpEvents")
	}
	if !strings.Contains(err.Error(), "action trail") {
		t.Errorf("expected error to mention action trail, got %v", err)
	}
}

func TestHandleEventsReturnsUnsupported(t *testing.T) {
	driver := newDriver("http://example.invalid")
	if _, err := driver.HandleEvents(context.Background(), ""); err == nil {
		t.Fatalf("expected error from HandleEvents")
	}
}
