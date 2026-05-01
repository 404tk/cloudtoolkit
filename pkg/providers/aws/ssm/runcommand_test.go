package ssm

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
	driver := &Driver{Client: client, Region: "us-east-1"}
	driver.SetPollOptions(time.Millisecond, 5, func(time.Duration) {})
	return driver
}

func TestRunCommandSendsAndPolls(t *testing.T) {
	pollCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := r.Header.Get("X-Amz-Target")
		switch target {
		case "AmazonSSM.SendCommand":
			if got := r.Header.Get("Content-Type"); got != "application/x-amz-json-1.1" {
				t.Fatalf("unexpected content-type: %s", got)
			}
			_, _ = w.Write([]byte(`{"Command":{"CommandId":"cmd-1"}}`))
		case "AmazonSSM.GetCommandInvocation":
			pollCount++
			if pollCount == 1 {
				_, _ = w.Write([]byte(`{"Status":"InProgress"}`))
				return
			}
			_, _ = w.Write([]byte(`{"Status":"Success","StandardOutputContent":"hello\n"}`))
		default:
			t.Fatalf("unexpected target: %s", target)
		}
	}))
	defer server.Close()

	driver := newTestDriver(t, server.URL)
	output := driver.RunCommand("i-1", "linux", "echo hello")
	if output != "hello\n" {
		t.Fatalf("unexpected output: %q", output)
	}
	if pollCount != 2 {
		t.Fatalf("expected 2 polls, got %d", pollCount)
	}
}

func TestRunCommandRetriesInvocationNotFound(t *testing.T) {
	pollCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-Amz-Target") {
		case "AmazonSSM.SendCommand":
			_, _ = w.Write([]byte(`{"Command":{"CommandId":"cmd-2"}}`))
		case "AmazonSSM.GetCommandInvocation":
			pollCount++
			if pollCount == 1 {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"__type":"InvocationDoesNotExist","Message":"not yet"}`))
				return
			}
			_, _ = w.Write([]byte(`{"Status":"Success","StandardOutputContent":"ready"}`))
		}
	}))
	defer server.Close()

	driver := newTestDriver(t, server.URL)
	if got := driver.RunCommand("i-2", "linux", "id"); got != "ready" {
		t.Fatalf("expected output 'ready', got %q", got)
	}
	if pollCount != 2 {
		t.Fatalf("expected 2 polls (one InvocationDoesNotExist + one Success), got %d", pollCount)
	}
}

func TestRunCommandTimeoutWhenStuck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-Amz-Target") {
		case "AmazonSSM.SendCommand":
			_, _ = w.Write([]byte(`{"Command":{"CommandId":"cmd-3"}}`))
		case "AmazonSSM.GetCommandInvocation":
			_, _ = w.Write([]byte(`{"Status":"InProgress"}`))
		}
	}))
	defer server.Close()

	driver := newTestDriver(t, server.URL)
	if got := driver.RunCommand("i-3", "linux", "sleep 1"); got != "" {
		t.Fatalf("expected empty output on timeout, got %q", got)
	}
}

func TestResolveDocumentName(t *testing.T) {
	cases := map[string]string{
		"":        api.SSMDocumentLinux,
		"linux":   api.SSMDocumentLinux,
		"LINUX":   api.SSMDocumentLinux,
		"windows": api.SSMDocumentWindows,
		"WINDOWS": api.SSMDocumentWindows,
	}
	for in, want := range cases {
		got, ok := resolveDocumentName(in)
		if !ok {
			t.Errorf("resolveDocumentName(%q) returned ok=false", in)
			continue
		}
		if got != want {
			t.Errorf("resolveDocumentName(%q) = %q, want %q", in, got, want)
		}
	}
	if _, ok := resolveDocumentName("freebsd"); ok {
		t.Errorf("expected unknown os to fail")
	}
}

func TestRunCommandRejectsBadOSType(t *testing.T) {
	driver := newTestDriver(t, "http://example.invalid")
	if got := driver.RunCommand("i-1", "freebsd", "uname"); got != "" {
		t.Fatalf("expected empty output for unsupported os, got %q", got)
	}
}

func TestIsInvocationNotFound(t *testing.T) {
	if !isInvocationNotFound(&api.APIError{Code: "InvocationDoesNotExist"}) {
		t.Errorf("expected match on Code field")
	}
	if isInvocationNotFound(&api.APIError{Code: "AccessDenied"}) {
		t.Errorf("expected no match on AccessDenied")
	}
	rawErr := &mockErr{msg: "InvocationDoesNotExist: not yet"}
	if !isInvocationNotFound(rawErr) {
		t.Errorf("expected substring match for raw error")
	}
	_ = strings.Contains // keep strings import live for raw err patterns
}

type mockErr struct{ msg string }

func (e *mockErr) Error() string { return e.msg }
