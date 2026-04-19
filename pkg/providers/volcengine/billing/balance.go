package billing

import (
	"context"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Client *api.Client
	Region string
}

func (d *Driver) QueryAccountBalance(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
	}
	if d.Client == nil {
		return
	}
	resp, err := d.Client.QueryBalanceAcct(ctx, d.requestRegion())
	if err == nil && resp.Result.AvailableBalance != "" {
		logger.Warning("Available cash amount:", resp.Result.AvailableBalance)
	}
}

func (d *Driver) requestRegion() string {
	if d.Region == "" || d.Region == "all" {
		return api.DefaultRegion
	}
	return d.Region
}
