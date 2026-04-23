package assistant

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/auth"
)

func TestDriverRunCommandHappyPathDecodesOutput(t *testing.T) {
	var actions []string
	polls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/regions/cn-north-1/createCommand":
			actions = append(actions, "createCommand")
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			var req api.CreateCommandRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("decode createCommand body: %v", err)
			}
			if req.RegionID != "cn-north-1" {
				t.Fatalf("unexpected region: %q", req.RegionID)
			}
			if req.CommandType != "shell" {
				t.Fatalf("unexpected command type: %q", req.CommandType)
			}
			if !strings.HasPrefix(req.CommandName, "ctk-") {
				t.Fatalf("unexpected command name: %q", req.CommandName)
			}
			want := base64.StdEncoding.EncodeToString([]byte("echo hello"))
			if req.CommandContent != want {
				t.Fatalf("unexpected commandContent: %q", req.CommandContent)
			}
			_, _ = w.Write([]byte(`{"requestId":"req-create","result":{"commandId":"cmd-1"}}`))
		case "/v1/regions/cn-north-1/invokeCommand":
			actions = append(actions, "invokeCommand")
			body, _ := io.ReadAll(r.Body)
			var req api.InvokeCommandRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("decode invokeCommand body: %v", err)
			}
			if req.CommandID != "cmd-1" || len(req.Instances) != 1 || req.Instances[0] != "i-1" {
				t.Fatalf("unexpected invoke payload: %+v", req)
			}
			_, _ = w.Write([]byte(`{"requestId":"req-invoke","result":{"invokeId":"inv-1"}}`))
		case "/v1/regions/cn-north-1/describeInvocations":
			actions = append(actions, "describeInvocations")
			polls++
			body, _ := io.ReadAll(r.Body)
			var req api.DescribeInvocationsRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("decode describeInvocations body: %v", err)
			}
			if len(req.InvokeIDs) != 1 || req.InvokeIDs[0] != "inv-1" {
				t.Fatalf("unexpected invokeIds: %+v", req.InvokeIDs)
			}
			if polls == 1 {
				_, _ = w.Write([]byte(`{"requestId":"req-poll-1","result":{"invocations":[{"status":"running","invokeId":"inv-1","invocationInstances":[{"instanceId":"i-1","status":"running"}]}]}}`))
				return
			}
			outputB64 := base64.StdEncoding.EncodeToString([]byte("hello\n"))
			_, _ = w.Write([]byte(`{"requestId":"req-poll-2","result":{"invocations":[{"status":"finish","invokeId":"inv-1","invocationInstances":[{"instanceId":"i-1","status":"finish","output":"` + outputB64 + `"}]}]}}`))
		case "/v1/regions/cn-north-1/deleteCommands":
			actions = append(actions, "deleteCommands")
			body, _ := io.ReadAll(r.Body)
			var req api.DeleteCommandsRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("decode deleteCommands body: %v", err)
			}
			if len(req.CommandIDs) != 1 || req.CommandIDs[0] != "cmd-1" {
				t.Fatalf("unexpected commandIds: %+v", req.CommandIDs)
			}
			_, _ = w.Write([]byte(`{"requestId":"req-delete","result":{"commandId":"cmd-1"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{
		Client:       newTestClient(server.URL),
		Region:       "cn-north-1",
		pollInterval: time.Millisecond,
		maxPolls:     3,
		sleep:        func(time.Duration) {},
	}
	if got := driver.RunCommand("i-1", "linux", "echo hello"); got != "hello\n" {
		t.Fatalf("unexpected output: %q", got)
	}
	if got := strings.Join(actions, ","); got != "createCommand,invokeCommand,describeInvocations,describeInvocations,deleteCommands" {
		t.Fatalf("unexpected action order: %s", got)
	}
}

func TestDriverRunCommandSurfacesInstanceErrorInfo(t *testing.T) {
	var actions []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/regions/cn-north-1/createCommand":
			actions = append(actions, "createCommand")
			_, _ = w.Write([]byte(`{"requestId":"req-create","result":{"commandId":"cmd-fail"}}`))
		case "/v1/regions/cn-north-1/invokeCommand":
			actions = append(actions, "invokeCommand")
			_, _ = w.Write([]byte(`{"requestId":"req-invoke","result":{"invokeId":"inv-fail"}}`))
		case "/v1/regions/cn-north-1/describeInvocations":
			actions = append(actions, "describeInvocations")
			_, _ = w.Write([]byte(`{"requestId":"req-poll","result":{"invocations":[{"status":"failed","invokeId":"inv-fail","invocationInstances":[{"instanceId":"i-1","status":"invalid","errorInfo":"cloud assistant agent offline"}]}]}}`))
		case "/v1/regions/cn-north-1/deleteCommands":
			actions = append(actions, "deleteCommands")
			_, _ = w.Write([]byte(`{"requestId":"req-delete","result":{"commandId":"cmd-fail"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{
		Client:       newTestClient(server.URL),
		Region:       "cn-north-1",
		pollInterval: time.Millisecond,
		maxPolls:     2,
		sleep:        func(time.Duration) {},
	}
	if got := driver.RunCommand("i-1", "linux", "bad"); got != "" {
		t.Fatalf("expected empty output on failure, got %q", got)
	}
	// deleteCommands must still run even when the invocation failed.
	if !strings.Contains(strings.Join(actions, ","), "deleteCommands") {
		t.Fatalf("deleteCommands cleanup missing, got: %v", actions)
	}
}

func TestDriverRunCommandRejectsEmptyRegion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("no HTTP request expected when region is empty: %s", r.URL.Path)
	}))
	defer server.Close()

	driver := &Driver{
		Client:       newTestClient(server.URL),
		Region:       "",
		pollInterval: time.Millisecond,
		maxPolls:     1,
		sleep:        func(time.Duration) {},
	}
	if got := driver.RunCommand("i-1", "linux", "echo"); got != "" {
		t.Fatalf("expected empty output, got %q", got)
	}
}

func TestDriverRunCommandRejectsUnknownOSType(t *testing.T) {
	var actions []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actions = append(actions, r.URL.Path)
	}))
	defer server.Close()

	driver := &Driver{
		Client:       newTestClient(server.URL),
		Region:       "cn-north-1",
		pollInterval: time.Millisecond,
		maxPolls:     1,
		sleep:        func(time.Duration) {},
	}
	if got := driver.RunCommand("i-1", "solaris", "uname -a"); got != "" {
		t.Fatalf("expected empty output, got %q", got)
	}
	if len(actions) != 0 {
		t.Fatalf("expected no API calls for unknown osType, got: %v", actions)
	}
}

func TestResolveCommandType(t *testing.T) {
	cases := []struct {
		osType string
		want   string
		ok     bool
	}{
		{"linux", "shell", true},
		{"LINUX", "shell", true},
		{" Windows ", "powershell", true},
		{"", "", false},
		{"freebsd", "", false},
	}
	for _, tc := range cases {
		got, ok := resolveCommandType(tc.osType)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("resolveCommandType(%q) = (%q, %v), want (%q, %v)", tc.osType, got, ok, tc.want, tc.ok)
		}
	}
}

func TestDecodeOutput(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"   ", ""},
		{base64.StdEncoding.EncodeToString([]byte("hello\n")), "hello\n"},
		{"not-valid-base64", "not-valid-base64"},
	}
	for _, tc := range cases {
		if got := decodeOutput(tc.in); got != tc.want {
			t.Fatalf("decodeOutput(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestInvocationErrorInfoPrefersTopLevelErrorInfoOverInstanceStatus(t *testing.T) {
	inv := api.Invocation{
		ErrorInfo: "agent offline",
		InvocationInstances: []api.InvocationInstance{
			{Status: "invalid"},
		},
	}
	if got := invocationErrorInfo(inv); got != "agent offline" {
		t.Fatalf("unexpected error info: %q", got)
	}
}

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
