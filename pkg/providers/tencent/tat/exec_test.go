package tat

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func TestRunCommandPollsUntilSuccessAndDecodesOutput(t *testing.T) {
	pollCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-TC-Action") {
		case "RunCommand":
			body := readBody(t, r)
			if body != `{"Content":"ZWNobyBoZWxsbw==","InstanceIds":["ins-1"],"CommandType":"SHELL"}` {
				t.Fatalf("unexpected RunCommand body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"CommandId":"cmd-1","InvocationId":"inv-1","RequestId":"req-run"}}`))
		case "DescribeInvocations":
			body := readBody(t, r)
			if body != `{"InvocationIds":["inv-1"]}` {
				t.Fatalf("unexpected DescribeInvocations body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"InvocationSet":[{"InvocationId":"inv-1","InvocationTaskBasicInfoSet":[{"InvocationTaskId":"task-1","TaskStatus":"RUNNING","InstanceId":"ins-1"}]}],"RequestId":"req-inv"}}`))
		case "DescribeInvocationTasks":
			pollCount++
			body := readBody(t, r)
			if body != `{"InvocationTaskIds":["task-1"],"HideOutput":false}` {
				t.Fatalf("unexpected DescribeInvocationTasks body: %s", body)
			}
			if pollCount == 1 {
				_, _ = w.Write([]byte(`{"Response":{"InvocationTaskSet":[{"InvocationId":"inv-1","InvocationTaskId":"task-1","TaskStatus":"RUNNING","TaskResult":{"Output":""}}],"RequestId":"req-task-1"}}`))
				return
			}
			encoded := base64.StdEncoding.EncodeToString([]byte("hello\n"))
			_, _ = w.Write([]byte(`{"Response":{"InvocationTaskSet":[{"InvocationId":"inv-1","InvocationTaskId":"task-1","TaskStatus":"SUCCESS","TaskResult":{"Output":"` + encoded + `"}}],"RequestId":"req-task-2"}}`))
		default:
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	output := driver.RunCommand("ins-1", "LINUX_UNIX", "echo hello")
	if output != "hello\n" {
		t.Fatalf("unexpected output: %q", output)
	}
	if pollCount != 2 {
		t.Fatalf("unexpected poll count: %d", pollCount)
	}
}

func TestRunCommandSupportsBase64PrefixAndWindowsCommandType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-TC-Action") {
		case "RunCommand":
			body := readBody(t, r)
			if body != `{"Content":"V3JpdGUtSG9zdCBoZWxsbw==","InstanceIds":["ins-2"],"CommandType":"POWERSHELL"}` {
				t.Fatalf("unexpected RunCommand body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"CommandId":"cmd-2","InvocationId":"inv-2","RequestId":"req-run"}}`))
		case "DescribeInvocations":
			_, _ = w.Write([]byte(`{"Response":{"InvocationSet":[{"InvocationId":"inv-2","InvocationTaskBasicInfoSet":[{"InvocationTaskId":"task-2","TaskStatus":"SUCCESS","InstanceId":"ins-2"}]}],"RequestId":"req-inv"}}`))
		case "DescribeInvocationTasks":
			encoded := base64.StdEncoding.EncodeToString([]byte("ok"))
			_, _ = w.Write([]byte(`{"Response":{"InvocationTaskSet":[{"InvocationId":"inv-2","InvocationTaskId":"task-2","TaskStatus":"SUCCESS","TaskResult":{"Output":"` + encoded + `"}}],"RequestId":"req-task"}}`))
		default:
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	output := driver.RunCommand("ins-2", "WINDOWS", "base64 V3JpdGUtSG9zdCBoZWxsbw==")
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
			switch r.Header.Get("X-TC-Action") {
			case "RunCommand":
				_, _ = w.Write([]byte(`{"Response":{"CommandId":"cmd-timeout","InvocationId":"inv-timeout","RequestId":"req-run"}}`))
			case "DescribeInvocations":
				_, _ = w.Write([]byte(`{"Response":{"InvocationSet":[{"InvocationId":"inv-timeout","InvocationTaskBasicInfoSet":[{"InvocationTaskId":"task-timeout","TaskStatus":"RUNNING","InstanceId":"ins-timeout"}]}],"RequestId":"req-inv"}}`))
			case "DescribeInvocationTasks":
				_, _ = w.Write([]byte(`{"Response":{"InvocationTaskSet":[{"InvocationId":"inv-timeout","InvocationTaskId":"task-timeout","TaskStatus":"RUNNING","TaskResult":{"Output":""}}],"RequestId":"req-task"}}`))
			default:
				t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
			}
		}))
		defer server.Close()

		driver := newTestDriver(server.URL)
		driver.maxPollAttempts = 2
		output := driver.RunCommand("ins-timeout", "LINUX_UNIX", "echo slow")
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
			switch r.Header.Get("X-TC-Action") {
			case "RunCommand":
				_, _ = w.Write([]byte(`{"Response":{"CommandId":"cmd-failed","InvocationId":"inv-failed","RequestId":"req-run"}}`))
			case "DescribeInvocations":
				_, _ = w.Write([]byte(`{"Response":{"InvocationSet":[{"InvocationId":"inv-failed","InvocationTaskBasicInfoSet":[{"InvocationTaskId":"task-failed","TaskStatus":"FAILED","InstanceId":"ins-failed"}]}],"RequestId":"req-inv"}}`))
			case "DescribeInvocationTasks":
				_, _ = w.Write([]byte(`{"Response":{"InvocationTaskSet":[{"InvocationId":"inv-failed","InvocationTaskId":"task-failed","TaskStatus":"FAILED","ErrorInfo":"exit status 1","TaskResult":{"Output":""}}],"RequestId":"req-task"}}`))
			default:
				t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
			}
		}))
		defer server.Close()

		driver := newTestDriver(server.URL)
		output := driver.RunCommand("ins-failed", "LINUX_UNIX", "exit 1")
		if output != "" {
			t.Fatalf("expected empty output, got %q", output)
		}
		if got := buffer.String(); !strings.Contains(got, "Exception status: FAILED") || !strings.Contains(got, "exit status 1") {
			t.Fatalf("unexpected logger output: %s", got)
		}
	})
}

func newTestDriver(baseURL string) Driver {
	return Driver{
		Credential: auth.New("ak", "sk", ""),
		Region:     "ap-guangzhou",
		clientOptions: []api.Option{
			api.WithBaseURL(baseURL),
			api.WithClock(func() time.Time { return time.Unix(1776458501, 0).UTC() }),
			api.WithRetryPolicy(api.RetryPolicy{
				MaxAttempts: 1,
				Sleep:       func(context.Context, time.Duration) error { return nil },
			}),
		},
		pollInterval:    time.Millisecond,
		maxPollAttempts: 5,
		sleep:           func(time.Duration) {},
	}
}

func readBody(t *testing.T, r *http.Request) string {
	t.Helper()
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(body)
}
