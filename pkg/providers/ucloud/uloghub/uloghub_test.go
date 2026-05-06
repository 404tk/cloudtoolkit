package uloghub

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
		Region:     "cn-bj2",
		Client: api.NewClient(credential,
			api.WithBaseURL(baseURL),
			api.WithRetryPolicy(api.RetryPolicy{MaxAttempts: 1}),
		),
	}
}

func TestGetLogsListsTopics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if got := r.Form.Get("Action"); got != "DescribeULogTopic" {
			t.Fatalf("unexpected action: %s", got)
		}
		_, _ = w.Write([]byte(`{"Action":"DescribeULogTopicResponse","RetCode":0,"TotalCount":2,"Topics":[
  {"TopicID":"tp-1","TopicName":"app","LogSetID":"ls-1","LogSetName":"prod","Region":"cn-bj2","CreateTime":1713427200,"UpdateTime":1713513600},
  {"TopicID":"tp-2","TopicName":"audit","LogSetID":"ls-1","LogSetName":"prod","Region":"cn-bj2","CreateTime":1713427260}
]}`))
	}))
	defer server.Close()

	driver := newDriver(server.URL)
	logs, err := driver.GetLogs(context.Background())
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 topics, got %d", len(logs))
	}
	if logs[0].ProjectName != "prod/app" {
		t.Errorf("expected logset/topic, got %q", logs[0].ProjectName)
	}
	if logs[0].LastModifyTime == "" {
		t.Errorf("expected formatted time, got empty")
	}
}

func TestGetLogsRejectsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Action":"DescribeULogTopicResponse","RetCode":171,"Message":"signature mismatch"}`))
	}))
	defer server.Close()

	driver := newDriver(server.URL)
	_, err := driver.GetLogs(context.Background())
	if err == nil {
		t.Fatal("expected error when DescribeULogTopic returns RetCode != 0")
	}
	if !strings.Contains(err.Error(), "signature mismatch") {
		t.Errorf("expected message in err, got %v", err)
	}
}

func TestGetLogsHandlesEmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Action":"DescribeULogTopicResponse","RetCode":0,"TotalCount":0,"Topics":[]}`))
	}))
	defer server.Close()

	driver := newDriver(server.URL)
	logs, err := driver.GetLogs(context.Background())
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 topics, got %d", len(logs))
	}
}
