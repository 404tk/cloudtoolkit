package sas

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func TestDumpEvents(t *testing.T) {
	logger.SetOutput(io.Discard)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("Action"); got != "DescribeSuspEvents" {
			t.Fatalf("unexpected action: %s", got)
		}
		_, _ = io.WriteString(w, `{"RequestId":"req-events","SuspEvents":[{"SecurityEventIds":"1,2","AlarmEventNameDisplay":"AccessKey leak","InstanceName":"ecs-1","EventStatus":2,"LastTime":"2026-04-18T12:00:00Z","Details":[{"NameDisplay":"调用的API","ValueDisplay":"ListBuckets"},{"NameDisplay":"调用IP","ValueDisplay":"1.1.1.1"},{"NameDisplay":"AK","ValueDisplay":"LTAI123"}]}]}`)
	}))
	defer server.Close()

	driver := Driver{
		Cred: aliauth.New("ak", "sk", ""),
		clientOptions: []api.Option{
			api.WithBaseURL(server.URL),
			api.WithClock(func() time.Time { return time.Unix(1713376800, 0).UTC() }),
			api.WithNonce(func() string { return "nonce" }),
		},
	}

	events, err := driver.DumpEvents(context.Background())
	if err != nil {
		t.Fatalf("DumpEvents() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("unexpected event count: %d", len(events))
	}
	event := events[0]
	if event.Id != "1,2" || event.Name != "AccessKey leak" || event.API != "ListBuckets" || event.SourceIp != "1.1.1.1" || event.AccessKey != "LTAI123" || event.Status != "已忽略" {
		t.Fatalf("unexpected event: %+v", event)
	}
}

func TestHandleEvents(t *testing.T) {
	logger.SetOutput(io.Discard)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("Action"); got != "HandleSecurityEvents" {
			t.Fatalf("unexpected action: %s", got)
		}
		if got := r.URL.Query().Get("OperationCode"); got != "advance_mark_mis_info" {
			t.Fatalf("unexpected operation code: %s", got)
		}
		if got := r.URL.Query().Get("SecurityEventIds.1"); got != "123" {
			t.Fatalf("unexpected first event id: %s", got)
		}
		if got := r.URL.Query().Get("SecurityEventIds.2"); got != "456" {
			t.Fatalf("unexpected second event id: %s", got)
		}
		_, _ = io.WriteString(w, `{"RequestId":"req-handle","HandleSecurityEventsResponse":{"TaskId":99}}`)
	}))
	defer server.Close()

	driver := Driver{
		Cred: aliauth.New("ak", "sk", ""),
		clientOptions: []api.Option{
			api.WithBaseURL(server.URL),
			api.WithClock(func() time.Time { return time.Unix(1713376800, 0).UTC() }),
			api.WithNonce(func() string { return "nonce" }),
		},
	}

	result, err := driver.HandleEvents(context.Background(), "123, 456")
	if err != nil {
		t.Fatalf("HandleEvents() error = %v", err)
	}
	if result.TaskID != 99 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = writer

	done := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(reader)
		done <- string(data)
	}()

	fn()

	_ = writer.Close()
	os.Stdout = originalStdout
	return <-done
}
