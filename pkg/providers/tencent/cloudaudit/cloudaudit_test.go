package cloudaudit

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
	d := &Driver{Credential: auth.New("ak", "sk", "")}
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

func TestDumpEventsMapsResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-TC-Action"); got != "LookUpEvents" {
			t.Fatalf("unexpected action: %s", got)
		}
		listOver := true
		_, _ = w.Write([]byte(`{"Response":{"ListOver":true,"Events":[{"EventId":"e1","EventName":"CreateUser","EventNameCn":"创建子用户","EventTime":"2026-04-22 09:10:11","EventRegion":"ap-guangzhou","Username":"alice","SourceIPAddress":"203.0.113.10","ResourceName":"ctk-demo-bot","Status":0,"SecretId":"AKID","ApiVersion":"2019-01-16"}],"RequestId":"r1"}}`))
		_ = listOver
	}))
	defer server.Close()

	driver := newTestDriver(t, server.URL)
	got, err := driver.DumpEvents(context.Background(), "")
	if err != nil {
		t.Fatalf("DumpEvents: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}
	ev := got[0]
	if ev.Id != "e1" || ev.API != "CreateUser" || ev.Name != "创建子用户" {
		t.Errorf("unexpected event: %+v", ev)
	}
	if ev.SourceIp != "203.0.113.10" || ev.AccessKey != "AKID" {
		t.Errorf("unexpected event detail: %+v", ev)
	}
	if ev.Status != "成功" {
		t.Errorf("expected status '成功', got %q", ev.Status)
	}
}

func TestDumpEventsPaginates(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch calls {
		case 1:
			_, _ = w.Write([]byte(`{"Response":{"ListOver":false,"NextToken":"page-2","Events":[{"EventId":"e1","EventName":"CreateUser"}],"RequestId":"r1"}}`))
		case 2:
			_, _ = w.Write([]byte(`{"Response":{"ListOver":true,"Events":[{"EventId":"e2","EventName":"DeleteUser"}],"RequestId":"r2"}}`))
		default:
			t.Fatalf("unexpected call: %d", calls)
		}
	}))
	defer server.Close()

	driver := newTestDriver(t, server.URL)
	got, err := driver.DumpEvents(context.Background(), "")
	if err != nil {
		t.Fatalf("DumpEvents: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 events across pages, got %d", len(got))
	}
}

func TestDumpEventsParsesTimeWindow(t *testing.T) {
	var sawBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 2048)
		n, _ := r.Body.Read(buf)
		sawBody = string(buf[:n])
		_, _ = w.Write([]byte(`{"Response":{"ListOver":true,"Events":[],"RequestId":"r1"}}`))
	}))
	defer server.Close()

	driver := newTestDriver(t, server.URL)
	if _, err := driver.DumpEvents(context.Background(), "1714694400:1714780800"); err != nil {
		t.Fatalf("DumpEvents: %v", err)
	}
	if !strings.Contains(sawBody, `"StartTime":1714694400`) || !strings.Contains(sawBody, `"EndTime":1714780800`) {
		t.Errorf("expected start/end time in body, got %s", sawBody)
	}
}

func TestDumpEventsRejectsMalformedWindow(t *testing.T) {
	driver := newTestDriver(t, "http://example.invalid")
	if _, err := driver.DumpEvents(context.Background(), "abc"); err == nil {
		t.Errorf("expected error for malformed window")
	}
}

func TestHandleEventsReturnsUnsupported(t *testing.T) {
	driver := newTestDriver(t, "http://example.invalid")
	_, err := driver.HandleEvents(context.Background(), "evt-1")
	if err == nil {
		t.Fatalf("expected unsupported error")
	}
	if !strings.Contains(err.Error(), "read-only") {
		t.Errorf("expected read-only error, got %v", err)
	}
}

func TestStatusLabel(t *testing.T) {
	cases := map[uint64]string{
		0: "成功",
		1: "失败",
		2: "部分失败",
		9: "",
	}
	for code, want := range cases {
		if got := statusLabel(code); got != want {
			t.Errorf("statusLabel(%d) = %q, want %q", code, got, want)
		}
	}
}
