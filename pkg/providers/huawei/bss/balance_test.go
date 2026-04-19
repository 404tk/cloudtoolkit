package bss

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func TestDriverQueryAccountBalanceUsesExpectedEndpointAndPrefersCashAccount(t *testing.T) {
	tests := []struct {
		name      string
		intl      bool
		wantHost  string
		wantValue string
	}{
		{name: "china", intl: false, wantHost: "bss.myhuaweicloud.com", wantValue: "66.80"},
		{name: "intl", intl: true, wantHost: "bss-intl.myhuaweicloud.com", wantValue: "88.90"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr := withLoggerBuffers(t)
			rt := &recordingTransport{
				t:        t,
				wantHost: tc.wantHost,
				body:     `{"account_balances":[{"account_type":2,"amount":"999.99"},{"account_type":1,"amount":"` + tc.wantValue + `"}]}`,
			}

			driver := &Driver{
				Cred: auth.New("AKID", "SECRET", "cn-north-4", tc.intl),
				Client: api.NewClient(
					auth.New("AKID", "SECRET", "cn-north-4", tc.intl),
					api.WithHTTPClient(&http.Client{Transport: rt}),
					api.WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
					api.WithRetryPolicy(noopRetryPolicy{}),
				),
			}

			driver.QueryAccountBalance(context.Background())

			if rt.gotPath != "/v2/accounts/customer-accounts/balances" {
				t.Fatalf("unexpected path: %s", rt.gotPath)
			}
			if !strings.Contains(stdout.String(), "Available cash amount: "+tc.wantValue) {
				t.Fatalf("unexpected stdout logs: %s", stdout.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("unexpected stderr logs: %s", stderr.String())
			}
		})
	}
}

type recordingTransport struct {
	t        *testing.T
	wantHost string
	body     string
	gotPath  string
}

func (rt *recordingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	rt.t.Helper()
	if r.URL.Host != rt.wantHost {
		rt.t.Fatalf("unexpected host: %s", r.URL.Host)
	}
	if r.Method != http.MethodGet {
		rt.t.Fatalf("unexpected method: %s", r.Method)
	}
	if got := r.Header.Get(api.HeaderAuthorization); got == "" {
		rt.t.Fatal("missing authorization header")
	}
	rt.gotPath = r.URL.Path
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(rt.body)),
		Request:    r,
	}, nil
}

type noopRetryPolicy struct{}

func (noopRetryPolicy) Do(ctx context.Context, _ bool, fn func() (*http.Response, error)) (*http.Response, error) {
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
