package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// AWS Cost Explorer GetCostAndUsage — JSON-1.1 RPC.
//
//	X-Amz-Target: AWSInsightsIndexService.GetCostAndUsage
//	Endpoint:     ce.us-east-1.amazonaws.com (global, signed for us-east-1)
//
// Cost Explorer has no native "account balance" concept — AWS bills monthly.
// We surface the unblended cost for the current calendar month, which is the
// closest equivalent and matches the project's "credit balance" semantics
// agreed in PLAN.md (see Task 7 / T2.1).
const (
	costExplorerContentType    = "application/x-amz-json-1.1"
	costExplorerGetCostAndUsage = "AWSInsightsIndexService.GetCostAndUsage"
)

type GetCostAndUsageInput struct {
	TimePeriod CostTimePeriod `json:"TimePeriod"`
	Granularity string        `json:"Granularity"`
	Metrics     []string      `json:"Metrics"`
}

type CostTimePeriod struct {
	Start string `json:"Start"`
	End   string `json:"End"`
}

type CostMetric struct {
	Amount string `json:"Amount"`
	Unit   string `json:"Unit"`
}

type CostResultByTime struct {
	TimePeriod CostTimePeriod         `json:"TimePeriod"`
	Total      map[string]CostMetric  `json:"Total"`
	Estimated  bool                   `json:"Estimated"`
}

type GetCostAndUsageOutput struct {
	ResultsByTime []CostResultByTime `json:"ResultsByTime"`
}

// CostExplorerCurrentMonthSpend returns the unblended cost for the current
// calendar month in USD. The first day of the month inclusive → today
// exclusive matches Cost Explorer's expected window.
func (c *Client) CostExplorerCurrentMonthSpend(ctx context.Context) (string, string, error) {
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	if now.Before(end) {
		end = now
	}
	input := GetCostAndUsageInput{
		TimePeriod: CostTimePeriod{
			Start: start.Format("2006-01-02"),
			End:   end.Format("2006-01-02"),
		},
		Granularity: "MONTHLY",
		Metrics:     []string{"UnblendedCost"},
	}
	body, err := json.Marshal(input)
	if err != nil {
		return "", "", err
	}
	headers := http.Header{}
	headers.Set("Content-Type", costExplorerContentType)
	headers.Set("X-Amz-Target", costExplorerGetCostAndUsage)
	var out GetCostAndUsageOutput
	if err := c.DoRESTJSON(ctx, Request{
		Service:    "ce",
		Region:     "us-east-1",
		Method:     http.MethodPost,
		Path:       "/",
		Body:       body,
		Headers:    headers,
		Idempotent: true,
	}, &out); err != nil {
		return "", "", err
	}
	if len(out.ResultsByTime) == 0 {
		return "", "USD", nil
	}
	metric, ok := out.ResultsByTime[0].Total["UnblendedCost"]
	if !ok {
		return "", "USD", nil
	}
	return metric.Amount, metric.Unit, nil
}
