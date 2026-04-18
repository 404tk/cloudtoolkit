package sls

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func TestListProjects(t *testing.T) {
	logger.SetOutput(io.Discard)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})

	driver := Driver{
		Cred:   aliauth.New("ak", "sk", "token"),
		Region: "cn-hangzhou",
		httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Host != "cn-hangzhou.log.aliyuncs.com" {
					t.Fatalf("unexpected host: %s", req.URL.Host)
				}
				if got := req.URL.Query().Get("offset"); got != "0" {
					t.Fatalf("unexpected offset: %s", got)
				}
				if got := req.URL.Query().Get("size"); got != "500" {
					t.Fatalf("unexpected size: %s", got)
				}
				body := `{"count":1,"total":1,"projects":[{"projectName":"ctk-log","region":"cn-hangzhou","description":"demo","lastModifyTime":"1713376800"}]}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(body)),
				}, nil
			}),
		},
	}

	logs, err := driver.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("unexpected log count: %d", len(logs))
	}
	if logs[0].ProjectName != "ctk-log" || logs[0].Region != "cn-hangzhou" || logs[0].Description != "demo" || logs[0].LastModifyTime == "" {
		t.Fatalf("unexpected logs: %+v", logs)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
