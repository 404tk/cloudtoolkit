// Package billing wraps AWS Cost Explorer for the cloudlist `balance` asset.
//
// AWS doesn't surface a "credit balance" — billing is monthly. We surface the
// unblended cost for the current calendar month, matching the cross-cloud
// "current month spend" semantics chosen in PLAN.md (Task 7 / T2.1).
package billing

import (
	"context"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Client *api.Client
}

// QueryAccountBalance reports the current-month unblended cost via logger,
// mirroring the alibaba/tencent/huawei balance drivers.
func (d *Driver) QueryAccountBalance(ctx context.Context) {
	if d == nil || d.Client == nil {
		return
	}
	select {
	case <-ctx.Done():
		return
	default:
	}
	amount, unit, err := d.Client.CostExplorerCurrentMonthSpend(ctx)
	if err != nil {
		// Cost Explorer often returns AccessDenied unless ce:GetCostAndUsage is
		// granted. Surface as Info, not a hard error, so cloudlist doesn't
		// fail just because billing data isn't available.
		logger.Info("AWS Cost Explorer query unavailable: " + err.Error())
		return
	}
	if amount == "" {
		return
	}
	logger.Warning("AWS current-month spend: " + amount + " " + unit)
}
