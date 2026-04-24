package billing

import (
	"context"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	ucloudauth "github.com/404tk/cloudtoolkit/pkg/providers/ucloud/auth"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Credential ucloudauth.Credential
	Client     *api.Client
	ProjectID  string
}

func (d *Driver) QueryAccountBalance(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	var resp api.GetBalanceResponse
	err := d.client().Do(ctx, api.Request{Action: "GetBalance"}, &resp)
	if err != nil {
		return
	}

	amount := strings.TrimSpace(resp.AccountInfo.AmountAvailable)
	if amount == "" {
		amount = strings.TrimSpace(resp.AccountInfo.Amount)
	}
	if amount != "" {
		logger.Warning("Available cash amount:", amount)
	}
}

func (d *Driver) client() *api.Client {
	if d.Client != nil {
		return d.Client
	}
	return api.NewClient(d.Credential, api.WithProjectID(d.ProjectID))
}
