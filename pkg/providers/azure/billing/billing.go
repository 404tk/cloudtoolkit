// Package billing wraps Azure Cost Management for the cloudlist `balance`
// asset. Azure has no native "credit balance" endpoint either; we surface the
// current month-to-date Cost (the closest equivalent across tenants) per the
// PLAN.md decision.
package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	azapi "github.com/404tk/cloudtoolkit/pkg/providers/azure/api"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Client          *azapi.Client
	SubscriptionIDs []string
}

// QueryAccountBalance reports current-month spend for each subscription via
// logger. Errors are surfaced as Info to keep cloudlist resilient when the
// caller credential lacks `Microsoft.CostManagement/query/action`.
func (d *Driver) QueryAccountBalance(ctx context.Context) {
	if d == nil || d.Client == nil {
		return
	}
	for _, sub := range d.SubscriptionIDs {
		sub = strings.TrimSpace(sub)
		if sub == "" {
			continue
		}
		amount, currency, err := d.querySubscription(ctx, sub)
		if err != nil {
			logger.Info(fmt.Sprintf("Azure Cost Management query unavailable for %s: %s", sub, err.Error()))
			continue
		}
		if amount == "" {
			continue
		}
		logger.Warning(fmt.Sprintf("Azure subscription %s month-to-date spend: %s %s", sub, amount, currency))
	}
}

func (d *Driver) querySubscription(ctx context.Context, subscription string) (string, string, error) {
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	body, err := json.Marshal(azapi.CostManagementQueryRequest{
		Type:      "ActualCost",
		Timeframe: "Custom",
		TimePeriod: &azapi.CostManagementQueryTimePeriod{
			From: start.Format("2006-01-02"),
			To:   now.Format("2006-01-02"),
		},
		Dataset: azapi.CostManagementQueryDataset{
			Granularity: "None",
			Aggregation: map[string]azapi.CostManagementAggregation{
				"totalCost": {Name: "Cost", Function: "Sum"},
			},
		},
	})
	if err != nil {
		return "", "", err
	}
	var resp azapi.CostManagementQueryResponse
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	if err := d.Client.Do(ctx, azapi.Request{
		Method: http.MethodPost,
		Path:   fmt.Sprintf("/subscriptions/%s/providers/Microsoft.CostManagement/query", subscription),
		Query: url.Values{
			"api-version": {azapi.CostManagementAPIVersion},
		},
		Headers: headers,
		Body:    body,
	}, &resp); err != nil {
		return "", "", err
	}
	return parseFirstRow(resp)
}

// parseFirstRow extracts the Cost + Currency cells from a Cost Management
// query response. With Granularity=None there is at most one row.
func parseFirstRow(resp azapi.CostManagementQueryResponse) (string, string, error) {
	if len(resp.Properties.Rows) == 0 {
		return "", "", nil
	}
	row := resp.Properties.Rows[0]
	costIdx, currencyIdx := -1, -1
	for i, col := range resp.Properties.Columns {
		switch strings.ToLower(col.Name) {
		case "cost", "totalcost":
			costIdx = i
		case "currency", "billingcurrency":
			currencyIdx = i
		}
	}
	if costIdx < 0 || costIdx >= len(row) {
		return "", "", nil
	}
	amount := formatCell(row[costIdx])
	currency := "USD"
	if currencyIdx >= 0 && currencyIdx < len(row) {
		if c := formatCell(row[currencyIdx]); c != "" {
			currency = c
		}
	}
	return amount, currency, nil
}

func formatCell(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		// Cost cells come back as JSON numbers; trim trailing zeros so the
		// printed value is readable.
		return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(t, 'f', 4, 64), "0"), ".")
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}
