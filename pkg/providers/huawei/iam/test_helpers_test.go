package iam

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func newTestDriver(baseURL, region string) *Driver {
	return &Driver{
		Cred: auth.New("AKID", "SECRET", region, false),
		Client: api.NewClient(
			auth.New("AKID", "SECRET", region, false),
			api.WithBaseURL(baseURL),
			api.WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
			api.WithRetryPolicy(testRetryPolicy{}),
		),
	}
}

type testRetryPolicy struct{}

func (testRetryPolicy) Do(ctx context.Context, _ bool, fn func() (*http.Response, error)) (*http.Response, error) {
	return fn()
}

func withLoggerBuffers(t *testing.T) (*bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	logger.SetOutputs(stdout, stderr)
	t.Cleanup(func() {
		logger.SetOutputs(os.Stdout, os.Stderr)
	})
	return stdout, stderr
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()

	_ = w.Close()
	os.Stdout = oldStdout
	output := <-done
	_ = r.Close()
	return output
}

func readBody(t *testing.T, r *http.Request) string {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	_ = r.Body.Close()
	return string(body)
}
