package billing

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func TestQueryAccountBalanceLogsAvailableCashAmount(t *testing.T) {
	original := &bytes.Buffer{}
	logger.SetOutput(original)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-TC-Action"); got != "DescribeAccountBalance" {
			t.Fatalf("unexpected action: %s", got)
		}
		_, _ = w.Write([]byte(`{"Response":{"RealBalance":12345,"RequestId":"req-1"}}`))
	}))
	defer server.Close()

	driver := Driver{
		Cred:   auth.New("ak", "sk", ""),
		Region: "all",
		clientOptions: []api.Option{
			api.WithBaseURL(server.URL),
			api.WithClock(func() time.Time { return time.Unix(1776458501, 0).UTC() }),
			api.WithRetryPolicy(api.RetryPolicy{
				MaxAttempts: 1,
				Sleep:       func(context.Context, time.Duration) error { return nil },
			}),
		},
	}

	driver.QueryAccountBalance(context.Background())

	if got := original.String(); !strings.Contains(got, "Available cash amount: 123.45") {
		t.Fatalf("unexpected logger output: %s", got)
	}
}
