package ecs

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func TestRunCommandPollsUntilFinished(t *testing.T) {
	pollCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("Action") {
		case "RunCommand":
			if got := r.URL.Query().Get("Type"); got != "RunShellScript" {
				t.Fatalf("unexpected command type: %s", got)
			}
			if got := r.URL.Query().Get("CommandContent"); got != "echo hello" {
				t.Fatalf("unexpected command content: %s", got)
			}
			if got := r.URL.Query().Get("InstanceId.1"); got != "i-1" {
				t.Fatalf("unexpected instance id: %s", got)
			}
			if got := r.URL.Query().Get("ContentEncoding"); got != "" {
				t.Fatalf("unexpected content encoding: %s", got)
			}
			_, _ = w.Write([]byte(`{"RequestId":"req-run","CommandId":"cmd-1","InvokeId":"inv-1"}`))
		case "DescribeInvocationResults":
			pollCount++
			if got := r.URL.Query().Get("CommandId"); got != "cmd-1" {
				t.Fatalf("unexpected command id: %s", got)
			}
			if got := r.URL.Query().Get("ContentEncoding"); got != "PlainText" {
				t.Fatalf("unexpected content encoding: %s", got)
			}
			if got := r.URL.Query().Get("PageSize"); got != "1" {
				t.Fatalf("unexpected page size: %s", got)
			}
			if pollCount == 1 {
				_, _ = w.Write([]byte(`{"RequestId":"req-poll-1","Invocation":{"InvocationResults":{"InvocationResult":[{"InvokeRecordStatus":"Running","Output":""}]}}}`))
				return
			}
			_, _ = w.Write([]byte(`{"RequestId":"req-poll-2","Invocation":{"InvocationResults":{"InvocationResult":[{"InvokeRecordStatus":"Finished","Output":"hello\n"}]}}}`))
		default:
			t.Fatalf("unexpected action: %s", r.URL.Query().Get("Action"))
		}
	}))
	defer server.Close()

	driver := newTestExecDriver(server.URL)
	output := driver.RunCommand("i-1", "linux", "echo hello")
	if output != "hello\n" {
		t.Fatalf("unexpected output: %q", output)
	}
	if pollCount != 2 {
		t.Fatalf("unexpected poll count: %d", pollCount)
	}
}

func TestRunCommandSupportsBase64PrefixAndWindowsCommandType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("Action") {
		case "RunCommand":
			if got := r.URL.Query().Get("Type"); got != "RunBatScript" {
				t.Fatalf("unexpected command type: %s", got)
			}
			if got := r.URL.Query().Get("CommandContent"); got != "V3JpdGUtSG9zdCBoZWxsbw==" {
				t.Fatalf("unexpected command content: %s", got)
			}
			if got := r.URL.Query().Get("ContentEncoding"); got != "Base64" {
				t.Fatalf("unexpected content encoding: %s", got)
			}
			_, _ = w.Write([]byte(`{"RequestId":"req-run","CommandId":"cmd-2","InvokeId":"inv-2"}`))
		case "DescribeInvocationResults":
			_, _ = w.Write([]byte(`{"RequestId":"req-poll","Invocation":{"InvocationResults":{"InvocationResult":[{"InvokeRecordStatus":"Finished","Output":"ok"}]}}}`))
		default:
			t.Fatalf("unexpected action: %s", r.URL.Query().Get("Action"))
		}
	}))
	defer server.Close()

	driver := newTestExecDriver(server.URL)
	output := driver.RunCommand("i-2", "windows", "base64 V3JpdGUtSG9zdCBoZWxsbw==")
	if output != "ok" {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRunCommandReturnsEmptyOnTimeoutAndFailure(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		logger.SetOutput(buffer)
		t.Cleanup(func() {
			logger.SetOutput(nil)
		})

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Query().Get("Action") {
			case "RunCommand":
				_, _ = w.Write([]byte(`{"RequestId":"req-run","CommandId":"cmd-timeout","InvokeId":"inv-timeout"}`))
			case "DescribeInvocationResults":
				_, _ = w.Write([]byte(`{"RequestId":"req-poll","Invocation":{"InvocationResults":{"InvocationResult":[{"InvokeRecordStatus":"Running","Output":""}]}}}`))
			default:
				t.Fatalf("unexpected action: %s", r.URL.Query().Get("Action"))
			}
		}))
		defer server.Close()

		driver := newTestExecDriver(server.URL)
		driver.maxPolls = 2
		output := driver.RunCommand("i-timeout", "linux", "echo slow")
		if output != "" {
			t.Fatalf("expected empty output, got %q", output)
		}
		if got := buffer.String(); !strings.Contains(got, "Timeout") {
			t.Fatalf("unexpected logger output: %s", got)
		}
	})

	t.Run("failed", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		logger.SetOutput(buffer)
		t.Cleanup(func() {
			logger.SetOutput(nil)
		})

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Query().Get("Action") {
			case "RunCommand":
				_, _ = w.Write([]byte(`{"RequestId":"req-run","CommandId":"cmd-failed","InvokeId":"inv-failed"}`))
			case "DescribeInvocationResults":
				_, _ = w.Write([]byte(`{"RequestId":"req-poll","Invocation":{"InvocationResults":{"InvocationResult":[{"InvokeRecordStatus":"Failed","Output":"","ErrorInfo":"exit status 1"}]}}}`))
			default:
				t.Fatalf("unexpected action: %s", r.URL.Query().Get("Action"))
			}
		}))
		defer server.Close()

		driver := newTestExecDriver(server.URL)
		output := driver.RunCommand("i-failed", "linux", "exit 1")
		if output != "" {
			t.Fatalf("expected empty output, got %q", output)
		}
		if got := buffer.String(); !strings.Contains(got, "Exception status: Failed") || !strings.Contains(got, "exit status 1") {
			t.Fatalf("unexpected logger output: %s", got)
		}
	})
}

func newTestExecDriver(baseURL string) Driver {
	return Driver{
		Cred:   aliauth.New("ak", "sk", ""),
		Region: "cn-hangzhou",
		clientOptions: []api.Option{
			api.WithBaseURL(baseURL),
			api.WithClock(func() time.Time { return time.Unix(1713376800, 0).UTC() }),
			api.WithNonce(func() string { return "nonce" }),
			api.WithRetryPolicy(api.RetryPolicy{
				MaxAttempts: 1,
				Sleep:       func(context.Context, time.Duration) error { return nil },
			}),
		},
		pollInterval: time.Millisecond,
		maxPolls:     5,
		sleep:        func(time.Duration) {},
	}
}
