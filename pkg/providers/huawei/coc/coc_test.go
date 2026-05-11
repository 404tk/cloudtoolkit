package coc

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

func testProjectCatalog() *api.ProjectCatalog {
	return api.NewProjectCatalog([]api.IAMProject{
		{ID: "project-1", Name: "cn-north-4"},
	}, "")
}

func jsonResponse(r *http.Request, body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    r,
	}
}

func TestExecuteSubmitsAndPolls(t *testing.T) {
	pollCount := 0
	deleteCount := 0
	driver := &Driver{
		Cred:           auth.New("AKID", "SECRET", "cn-north-4", false),
		Regions:        []string{"cn-north-4"},
		ProjectCatalog: testProjectCatalog(),
		nowSleep:       func(time.Duration) {},
		Client: newTestClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Host != "coc.myhuaweicloud.com" {
				t.Fatalf("unexpected COC host: %s", r.URL.Host)
			}
			if got := r.Header.Get("x-project-id"); got != "project-1" {
				t.Fatalf("expected x-project-id=project-1, got %q", got)
			}
			switch {
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/v1/job/scripts"):
				body, _ := io.ReadAll(r.Body)
				if !strings.Contains(string(body), `"type":"SHELL"`) {
					t.Fatalf("expected SHELL script create body, got %s", string(body))
				}
				return jsonResponse(r, `{"data":"script-1"}`), nil
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/v1/job/scripts/script-1"):
				return jsonResponse(r, `{"data":"exec-1"}`), nil
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/v1/job/script/orders/exec-1"):
				pollCount++
				if pollCount < 2 {
					return jsonResponse(r, `{"data":{"execute_uuid":"exec-1","status":"PROCESSING","properties":{"current_execute_batch_index":1}}}`), nil
				}
				return jsonResponse(r, `{"data":{"execute_uuid":"exec-1","status":"FINISHED","properties":{"current_execute_batch_index":1}}}`), nil
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/v1/job/script/orders/exec-1/batches/1"):
				return jsonResponse(r, `{"data":{"batch_index":1,"total_instances":1,"execute_instances":[{"target_instance":{"resource_id":"vm-1","provider":"ECS","region_id":"cn-north-4","type":"CLOUDSERVER"},"status":"FINISHED","message":"hello world"}]}}`), nil
			case r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "/v1/job/scripts/script-1"):
				deleteCount++
				return jsonResponse(r, `{"data":"script-1"}`), nil
			default:
				t.Fatalf("unexpected request: %s %s%s", r.Method, r.URL.Host, r.URL.Path)
				return nil, nil
			}
		})),
	}
	res, err := driver.Execute(context.Background(), "vm-1", "echo hello")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(res.Output, "hello world") {
		t.Errorf("expected output to include script stdout, got %q", res.Output)
	}
	if !strings.Contains(res.Output, "FINISHED") {
		t.Errorf("expected status header, got %q", res.Output)
	}
	if pollCount < 2 {
		t.Errorf("expected at least 2 polls (waiting for completion), got %d", pollCount)
	}
	if deleteCount != 1 {
		t.Errorf("expected best-effort script delete, got %d", deleteCount)
	}
}

func TestExecuteOSWindowsUsesBATScript(t *testing.T) {
	captured := ""
	driver := &Driver{
		Cred:           auth.New("AKID", "SECRET", "cn-north-4", false),
		Regions:        []string{"cn-north-4"},
		ProjectCatalog: testProjectCatalog(),
		nowSleep:       func(time.Duration) {},
		Client: newTestClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/v1/job/scripts") {
				body, _ := io.ReadAll(r.Body)
				captured = string(body)
				return jsonResponse(r, `{"data":""}`), nil
			}
			t.Fatalf("unexpected request: %s %s%s", r.Method, r.URL.Host, r.URL.Path)
			return nil, nil
		})),
	}
	_, err := driver.ExecuteOS(context.Background(), "vm-1", "windows", "whoami")
	if err == nil {
		t.Fatal("expected empty script uuid error")
	}
	if !strings.Contains(captured, `"type":"BAT"`) {
		t.Fatalf("expected BAT script create body, got %s", captured)
	}
	if !strings.Contains(captured, "@echo off") || !strings.Contains(captured, "whoami") {
		t.Fatalf("expected batch wrapper around command, got %s", captured)
	}
	if strings.Contains(captured, "/bin/bash") {
		t.Fatalf("windows script should not use bash wrapper, got %s", captured)
	}
}

func TestExecuteRejectsEmptyArgs(t *testing.T) {
	driver := &Driver{Cred: auth.New("AKID", "SECRET", "cn-north-4", false)}
	if _, err := driver.Execute(context.Background(), "", "echo hi"); err == nil {
		t.Error("expected error for empty instance id")
	}
	if _, err := driver.Execute(context.Background(), "vm-1", " "); err == nil {
		t.Error("expected error for empty command")
	}
}

func TestExecutePropagatesSubmitError(t *testing.T) {
	driver := &Driver{
		Cred:           auth.New("AKID", "SECRET", "cn-north-4", false),
		Regions:        []string{"cn-north-4"},
		ProjectCatalog: testProjectCatalog(),
		nowSleep:       func(time.Duration) {},
		Client: newTestClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"error_code":"COC.0403","error_msg":"forbidden"}`)),
				Request:    r,
			}, nil
		})),
	}
	_, err := driver.Execute(context.Background(), "vm-1", "echo hi")
	if err == nil {
		t.Fatal("expected error when submission fails")
	}
	if !strings.Contains(err.Error(), "COC.0403") {
		t.Errorf("expected COC.0403 in err, got %v", err)
	}
}

func TestWrapShellAddsShebang(t *testing.T) {
	wrapped := wrapShell("echo hi")
	if !strings.HasPrefix(wrapped, "#!/bin/bash") {
		t.Errorf("expected shebang prefix, got %q", wrapped)
	}
	if wrapShell("#!/usr/bin/env python\nprint('hi')") != "#!/usr/bin/env python\nprint('hi')" {
		t.Error("expected explicit shebang to be preserved")
	}
}
