package lts

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

type noopRetryPolicy struct{}

func (noopRetryPolicy) Do(ctx context.Context, _ bool, fn func() (*http.Response, error)) (*http.Response, error) {
	return fn()
}

func newTestClient(t *testing.T, fn roundTripFunc) *api.Client {
	t.Helper()
	return api.NewClient(
		auth.New("AKID", "SECRET", "cn-north-4", false),
		api.WithHTTPClient(&http.Client{Transport: fn}),
		api.WithRetryPolicy(noopRetryPolicy{}),
		api.WithClock(func() time.Time { return time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC) }),
	)
}

func jsonResponse(r *http.Request, body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    r,
	}
}

func TestGetLogsListsLogGroups(t *testing.T) {
	driver := &Driver{
		Cred:     auth.New("AKID", "SECRET", "cn-north-4", false),
		Regions:  []string{"cn-north-4"},
		DomainID: "d-1",
		Client: newTestClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.URL.Host == "iam.cn-north-4.myhuaweicloud.com" && r.URL.Path == "/v3/projects":
				return jsonResponse(r, `{"projects":[{"id":"project-n4","name":"cn-north-4","domain_id":"d-1","enabled":true}]}`), nil
			case r.URL.Host == "lts.cn-north-4.myhuaweicloud.com" && r.URL.Path == "/v2/project-n4/groups":
				return jsonResponse(r, `{"log_groups":[
  {"log_group_id":"lg-1","log_group_name":"prod-app","creation_time":1713427200000,"ttl_in_days":30},
  {"log_group_id":"lg-2","log_group_name":"audit-logs","creation_time":1713427260000,"ttl_in_days":90}
]}`), nil
			default:
				t.Fatalf("unexpected request: %s %s%s", r.Method, r.URL.Host, r.URL.Path)
				return nil, nil
			}
		})),
	}
	logs, err := driver.GetLogs(context.Background())
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 log groups, got %d", len(logs))
	}
	if logs[0].ProjectName != "prod-app" || logs[0].Region != "cn-north-4" {
		t.Errorf("unexpected first log: %+v", logs[0])
	}
	if logs[0].LastModifyTime == "" {
		t.Errorf("expected formatted creation time, got empty")
	}
	if logs[0].Description != "lg-1" {
		t.Errorf("expected log_group_id in description, got %q", logs[0].Description)
	}
}

func TestGetLogsRejectsAccessDenied(t *testing.T) {
	driver := &Driver{
		Cred:     auth.New("AKID", "SECRET", "cn-north-4", false),
		Regions:  []string{"cn-north-4"},
		DomainID: "d-1",
		Client: newTestClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path == "/v3/projects" {
				return jsonResponse(r, `{"projects":[{"id":"project-n4","name":"cn-north-4","domain_id":"d-1","enabled":true}]}`), nil
			}
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"error_code":"LTS.0403","error_msg":"forbidden"}`)),
				Request:    r,
			}, nil
		})),
	}
	_, err := driver.GetLogs(context.Background())
	if err == nil {
		t.Fatal("expected error when ListLogGroups fails")
	}
	if !strings.Contains(err.Error(), "LTS.0403") {
		t.Errorf("expected LTS.0403 in err, got %v", err)
	}
}

func TestGetLogsHandlesEmptyResults(t *testing.T) {
	driver := &Driver{
		Cred:     auth.New("AKID", "SECRET", "cn-north-4", false),
		Regions:  []string{"cn-north-4"},
		DomainID: "d-1",
		Client: newTestClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path == "/v3/projects" {
				return jsonResponse(r, `{"projects":[{"id":"project-n4","name":"cn-north-4","domain_id":"d-1","enabled":true}]}`), nil
			}
			return jsonResponse(r, `{"log_groups":[]}`), nil
		})),
	}
	logs, err := driver.GetLogs(context.Background())
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 log groups, got %d", len(logs))
	}
}
