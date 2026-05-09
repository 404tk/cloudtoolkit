package cloudaudit

import (
	"context"
	"fmt"
	"io"
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
	clock := func() time.Time { return time.Unix(1776458501, 0).UTC() }
	d := &Driver{Credential: auth.New("AKIDCURRENT", "sk", ""), Clock: clock}
	d.SetClientOptions(
		api.WithBaseURL(baseURL),
		api.WithClock(clock),
		api.WithRetryPolicy(api.RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)
	return d
}

func TestDumpEventsPaginates(t *testing.T) {
	calls := 0
	var secondBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch calls {
		case 1:
			_, _ = w.Write([]byte(`{"Response":{"ListOver":false,"NextToken":2,"Events":[{"EventId":"e1","EventName":"CreateUser"}],"RequestId":"r1"}}`))
		case 2:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			secondBody = string(body)
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
	if !strings.Contains(secondBody, `"NextToken":"2"`) {
		t.Fatalf("expected numeric response token to be sent as string, got %s", secondBody)
	}
}

func TestDumpEventsCapsDefaultOutputAndFormatsUnixTime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body strings.Builder
		body.WriteString(`{"Response":{"ListOver":true,"Events":[`)
		for i := 0; i < defaultEventLimit+5; i++ {
			if i > 0 {
				body.WriteByte(',')
			}
			_, _ = fmt.Fprintf(&body, `{"EventId":"e%d","EventName":"CreateUser","EventTime":"1778317184"}`, i)
		}
		body.WriteString(`],"RequestId":"r1"}}`)
		_, _ = w.Write([]byte(body.String()))
	}))
	defer server.Close()

	driver := newTestDriver(t, server.URL)
	got, err := driver.DumpEvents(context.Background(), "")
	if err != nil {
		t.Fatalf("DumpEvents: %v", err)
	}
	if len(got) != defaultEventLimit {
		t.Fatalf("expected %d events, got %d", defaultEventLimit, len(got))
	}
	if got[defaultEventLimit-1].Id != "e19" {
		t.Fatalf("expected output to stop at e19, got %s", got[defaultEventLimit-1].Id)
	}
	wantTime := time.Unix(1778317184, 0).UTC().Format(time.RFC3339)
	if got[0].Time != wantTime {
		t.Fatalf("expected formatted event time %q, got %q", wantTime, got[0].Time)
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
