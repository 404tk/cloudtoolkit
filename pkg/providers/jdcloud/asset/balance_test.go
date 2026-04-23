package asset

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/auth"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func TestDriverQueryAccountBalanceUsesExpectedEndpoint(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	logger.SetOutputs(stdout, stderr)
	t.Cleanup(func() {
		logger.SetOutputs(os.Stdout, os.Stderr)
	})

	rt := &recordingTransport{t: t}
	driver := &Driver{
		Client: api.NewClient(
			auth.New("AKID", "SECRET", "token64"),
			api.WithHTTPClient(&http.Client{Transport: rt}),
			api.WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
			api.WithNonceFunc(func() string { return "ebf8b26d-c3be-402f-9f10-f8b6573fd823" }),
			api.WithRetryPolicy(api.RetryPolicy{
				MaxAttempts: 1,
				Sleep:       func(context.Context, time.Duration) error { return nil },
			}),
		),
		Region: "all",
	}

	driver.QueryAccountBalance(context.Background())

	if rt.gotHost != "asset.jdcloud-api.com" {
		t.Fatalf("unexpected host: %s", rt.gotHost)
	}
	if rt.gotPath != "/v1/regions/cn-north-1/assets:describeAccountAmount" {
		t.Fatalf("unexpected path: %s", rt.gotPath)
	}
	if !strings.Contains(stdout.String(), "Available cash amount: 66.80") {
		t.Fatalf("unexpected stdout logs: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr logs: %s", stderr.String())
	}
}

type recordingTransport struct {
	t       *testing.T
	gotHost string
	gotPath string
}

func (rt *recordingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	rt.t.Helper()
	rt.gotHost = r.Host
	rt.gotPath = r.URL.Path
	if r.Method != http.MethodGet {
		rt.t.Fatalf("unexpected method: %s", r.Method)
	}
	if got := r.Header.Get(api.HeaderAuthorization); got == "" {
		rt.t.Fatal("missing authorization header")
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"requestId":"req-balance","result":{"availableAmount":"66.80","totalAmount":"88.90"}}`)),
		Request:    r,
	}, nil
}
