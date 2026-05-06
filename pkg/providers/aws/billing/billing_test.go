package billing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/aws/auth"
)

func newClient(baseURL string) *api.Client {
	return api.NewClient(
		auth.New("AKID", "SECRET", ""),
		api.WithBaseURL(baseURL),
		api.WithClock(func() time.Time { return time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC) }),
		api.WithRetryPolicy(api.RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)
}

func TestCostExplorerCurrentMonthSpendParses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Amz-Target"); got != "AWSInsightsIndexService.GetCostAndUsage" {
			t.Errorf("unexpected target: %s", got)
		}
		_, _ = w.Write([]byte(`{"ResultsByTime":[{"TimePeriod":{"Start":"2026-04-01","End":"2026-04-18"},"Total":{"UnblendedCost":{"Amount":"123.45","Unit":"USD"}},"Estimated":true}]}`))
	}))
	defer server.Close()

	amount, unit, err := newClient(server.URL).CostExplorerCurrentMonthSpend(context.Background())
	if err != nil {
		t.Fatalf("CostExplorerCurrentMonthSpend: %v", err)
	}
	if amount != "123.45" || unit != "USD" {
		t.Fatalf("unexpected result: %q %q", amount, unit)
	}
}

func TestQueryAccountBalanceSwallowsAccessDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"__type":"AccessDeniedException","Message":"User is not authorized"}`, http.StatusBadRequest)
	}))
	defer server.Close()

	driver := &Driver{Client: newClient(server.URL)}
	// Should not panic or propagate the error — balance is best-effort.
	driver.QueryAccountBalance(context.Background())
}

func TestQueryAccountBalanceWithEmptyResultsIsNoOp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ResultsByTime":[]}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newClient(server.URL)}
	driver.QueryAccountBalance(context.Background())
}

func TestCostExplorerSendsExpectedTimeWindow(t *testing.T) {
	var sawBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		sawBody = string(buf)
		_, _ = w.Write([]byte(`{"ResultsByTime":[{"Total":{"UnblendedCost":{"Amount":"0","Unit":"USD"}}}]}`))
	}))
	defer server.Close()

	_, _, err := newClient(server.URL).CostExplorerCurrentMonthSpend(context.Background())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(sawBody, `"Granularity":"MONTHLY"`) {
		t.Errorf("expected MONTHLY granularity in body, got: %s", sawBody)
	}
	if !strings.Contains(sawBody, `"UnblendedCost"`) {
		t.Errorf("expected UnblendedCost metric in body, got: %s", sawBody)
	}
}
