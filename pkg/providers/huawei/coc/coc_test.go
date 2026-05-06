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
	driver := &Driver{
		Cred:     auth.New("AKID", "SECRET", "cn-north-4", false),
		Regions:  []string{"cn-north-4"},
		nowSleep: func(time.Duration) {},
		Client: newTestClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/v1/job/scripts/orders/batch-execute"):
				return jsonResponse(r, `{"order_id":"ord-1","job_id":"job-1","status":"PENDING"}`), nil
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/orders/ord-1"):
				pollCount++
				if pollCount < 2 {
					return jsonResponse(r, `{"order_id":"ord-1","status":"RUNNING"}`), nil
				}
				return jsonResponse(r, `{"order_id":"ord-1","status":"FINISHED","instances":[{"instance_id":"vm-1","execute_status":"SUCCESS","output":"hello world"}]}`), nil
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
	if !strings.Contains(res.Output, "SUCCESS") {
		t.Errorf("expected status header, got %q", res.Output)
	}
	if pollCount < 2 {
		t.Errorf("expected at least 2 polls (waiting for completion), got %d", pollCount)
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
		Cred:     auth.New("AKID", "SECRET", "cn-north-4", false),
		Regions:  []string{"cn-north-4"},
		nowSleep: func(time.Duration) {},
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
