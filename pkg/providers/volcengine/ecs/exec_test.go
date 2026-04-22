package ecs

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestDriverRunCommandPollsUntilSuccessAndDecodesOutput(t *testing.T) {
	polls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			t.Fatalf("parse query: %v", err)
		}
		switch values.Get("Action") {
		case "DescribeCloudAssistantStatus":
			if values.Get("InstanceIds.1") != "i-1" {
				t.Fatalf("unexpected instance id: %s", values.Get("InstanceIds.1"))
			}
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-status"},"Result":{"Instances":[{"InstanceId":"i-1","Status":"Running","ClientVersion":"1.0.0","LastHeartbeatTime":"2026-04-22T12:00:00Z"}],"PageNumber":1,"PageSize":20,"TotalCount":1}}`))
		case "CreateCommand":
			if r.Method != http.MethodGet {
				t.Fatalf("unexpected method: %s", r.Method)
			}
			if values.Get("Name") == "" {
				t.Fatal("expected command name")
			}
			if values.Get("Type") != "Shell" {
				t.Fatalf("unexpected type: %s", values.Get("Type"))
			}
			if values.Get("ContentEncoding") != "Base64" {
				t.Fatalf("unexpected content encoding: %s", values.Get("ContentEncoding"))
			}
			if values.Get("CommandContent") != base64.StdEncoding.EncodeToString([]byte("echo hello")) {
				t.Fatalf("unexpected command content: %s", values.Get("CommandContent"))
			}
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-create"},"Result":{"CommandId":"cmd-1"}}`))
		case "InvokeCommand":
			if values.Get("CommandId") != "cmd-1" {
				t.Fatalf("unexpected command id: %s", values.Get("CommandId"))
			}
			if values.Get("InvocationName") == "" {
				t.Fatal("expected invocation name")
			}
			if values.Get("InstanceIds.1") != "i-1" {
				t.Fatalf("unexpected instance id: %s", values.Get("InstanceIds.1"))
			}
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-invoke"},"Result":{"InvocationId":"inv-1"}}`))
		case "DescribeInvocationResults":
			if values.Get("InvocationId") != "inv-1" || values.Get("CommandId") != "cmd-1" || values.Get("InstanceId") != "i-1" {
				t.Fatalf("unexpected describe args: %s", r.URL.RawQuery)
			}
			polls++
			if polls == 1 {
				_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-poll-1"},"Result":{"InvocationResults":[{"InvocationId":"inv-1","CommandId":"cmd-1","InstanceId":"i-1","InvocationResultStatus":"Running"}]}}`))
				return
			}
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-poll-2"},"Result":{"InvocationResults":[{"InvocationId":"inv-1","CommandId":"cmd-1","InstanceId":"i-1","InvocationResultStatus":"Success","Output":"aGVsbG8K"}]}}`))
		case "DeleteCommand":
			if values.Get("CommandId") != "cmd-1" {
				t.Fatalf("unexpected delete command id: %s", values.Get("CommandId"))
			}
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-delete"},"Result":{"CommandId":"cmd-1"}}`))
		default:
			t.Fatalf("unexpected action: %s", values.Get("Action"))
		}
	}))
	defer server.Close()

	driver := &Driver{
		Client:       newTestClient(server.URL),
		Region:       "cn-beijing",
		pollInterval: time.Millisecond,
		maxPolls:     3,
		sleep:        func(time.Duration) {},
	}
	got := driver.RunCommand("i-1", "Linux", "echo hello")
	if got != "hello\n" {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestDriverRunCommandReturnsEmptyOnFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			t.Fatalf("parse query: %v", err)
		}
		switch values.Get("Action") {
		case "DescribeCloudAssistantStatus":
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-status"},"Result":{"Instances":[{"InstanceId":"i-1","Status":"Running","ClientVersion":"1.0.0","LastHeartbeatTime":"2026-04-22T12:00:00Z"}],"PageNumber":1,"PageSize":20,"TotalCount":1}}`))
		case "CreateCommand":
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-create"},"Result":{"CommandId":"cmd-failed"}}`))
		case "InvokeCommand":
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-invoke"},"Result":{"InvocationId":"inv-failed"}}`))
		case "DescribeInvocationResults":
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-poll"},"Result":{"InvocationResults":[{"InvocationId":"inv-failed","CommandId":"cmd-failed","InstanceId":"i-1","InvocationResultStatus":"Failed","ErrorCode":"CommandFailed","ErrorMessage":"exit status 1"}]}}`))
		case "DeleteCommand":
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-delete"},"Result":{"CommandId":"cmd-failed"}}`))
		default:
			t.Fatalf("unexpected action: %s", values.Get("Action"))
		}
	}))
	defer server.Close()

	driver := &Driver{
		Client:       newTestClient(server.URL),
		Region:       "cn-beijing",
		pollInterval: time.Millisecond,
		maxPolls:     2,
		sleep:        func(time.Duration) {},
	}
	if got := driver.RunCommand("i-1", "Linux", "exit 1"); got != "" {
		t.Fatalf("expected empty output, got %q", got)
	}
}

func TestDriverRunCommandRequiresRunningCloudAssistant(t *testing.T) {
	var actions []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			t.Fatalf("parse query: %v", err)
		}
		actions = append(actions, values.Get("Action"))
		switch values.Get("Action") {
		case "DescribeCloudAssistantStatus":
			_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-status"},"Result":{"Instances":[{"InstanceId":"i-1","Status":"Initializing"}],"PageNumber":1,"PageSize":20,"TotalCount":1}}`))
		default:
			t.Fatalf("unexpected action: %s", values.Get("Action"))
		}
	}))
	defer server.Close()

	driver := &Driver{
		Client:       newTestClient(server.URL),
		Region:       "cn-beijing",
		pollInterval: time.Millisecond,
		maxPolls:     2,
		sleep:        func(time.Duration) {},
	}
	if got := driver.RunCommand("i-1", "Linux", "id"); got != "" {
		t.Fatalf("expected empty output, got %q", got)
	}
	if got := strings.Join(actions, ","); got != "DescribeCloudAssistantStatus" {
		t.Fatalf("unexpected actions: %s", got)
	}
}

func TestDriverGetResourceCachesHosts(t *testing.T) {
	SetCacheHostList(nil)
	defer SetCacheHostList(nil)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			t.Fatalf("parse query: %v", err)
		}
		switch values.Get("Action") {
		case "DescribeInstances":
			_, _ = w.Write(mustJSON(t, describeInstancesWithIDs([]string{"cache-1"}, "")))
		default:
			t.Fatalf("unexpected action: %s", values.Get("Action"))
		}
	}))
	defer server.Close()

	driver := &Driver{
		Client: newTestClient(server.URL),
		Region: "cn-beijing",
	}
	got, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("unexpected hosts: %+v", got)
	}

	cached := GetCacheHostList()
	if len(cached) != 1 || cached[0].ID != "cache-1" {
		t.Fatalf("unexpected cached hosts: %+v", cached)
	}
}
