package bss

import (
	"context"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Cred          aliauth.Credential
	Region        string
	clientOptions []api.Option
}

func (d *Driver) newClient() *api.Client {
	return api.NewClient(d.Cred, d.clientOptions...)
}

func (d *Driver) QueryAccountBalance(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
	}
	resp, err := d.newClient().QueryAccountBalance(ctx, api.NormalizeRegion(d.Region))
	if err == nil {
		if resp.Data.AvailableCashAmount != "" {
			logger.Warning("Available cash amount:", resp.Data.AvailableCashAmount)
		}
	}
}
