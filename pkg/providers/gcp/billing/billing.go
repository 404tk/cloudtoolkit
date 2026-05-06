// Package billing wraps GCP Cloud Billing for the cloudlist `balance` asset.
//
// GCP doesn't expose a credit balance via API; the closest management-plane
// signal is the list of billing accounts the caller's principal can see.
// Each open account corresponds to a paid project bucket — CSPM detectors
// use this to flag accounts unexpectedly disabled / detached.
package billing

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const (
	pageSize = 50
	maxPages = 20
)

type Driver struct {
	Client *api.Client
}

// QueryAccountBalance prints the visible billing accounts via logger. Errors
// are surfaced as Info to keep cloudlist resilient when the caller credential
// lacks `billing.accounts.list`.
func (d *Driver) QueryAccountBalance(ctx context.Context) {
	if d == nil || d.Client == nil {
		return
	}
	accounts, err := d.list(ctx)
	if err != nil {
		logger.Info("GCP Cloud Billing query unavailable: " + err.Error())
		return
	}
	if len(accounts) == 0 {
		return
	}
	open := 0
	for _, a := range accounts {
		if a.Open {
			open++
		}
	}
	logger.Warning(fmt.Sprintf("GCP billing accounts visible: %d (open=%d)", len(accounts), open))
}

func (d *Driver) list(ctx context.Context) ([]api.BillingAccount, error) {
	out := []api.BillingAccount{}
	pageToken := ""
	for page := 0; page < maxPages; page++ {
		query := url.Values{}
		query.Set("pageSize", strconv.Itoa(pageSize))
		if pageToken != "" {
			query.Set("pageToken", pageToken)
		}
		var resp api.ListBillingAccountsResponse
		if err := d.Client.Do(ctx, api.Request{
			Method:     http.MethodGet,
			BaseURL:    api.CloudBillingBaseURL,
			Path:       "/v1/billingAccounts",
			Query:      query,
			Idempotent: true,
		}, &resp); err != nil {
			return out, err
		}
		out = append(out, resp.BillingAccounts...)
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return out, nil
}
