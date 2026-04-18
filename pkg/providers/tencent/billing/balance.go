package billing

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Cred          auth.Credential
	Region        string
	clientOptions []api.Option
}

func (d *Driver) newClient() *api.Client {
	return api.NewClient(d.Cred, d.clientOptions...)
}

func (d *Driver) SetClientOptions(opts ...api.Option) {
	d.clientOptions = append([]api.Option(nil), opts...)
}

func (d *Driver) QueryAccountBalance(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
	}
	resp, err := d.newClient().DescribeAccountBalance(ctx, d.Region)
	if err == nil {
		var realBalance float64
		switch {
		case resp.Response.RealBalance != nil:
			realBalance = *resp.Response.RealBalance
		case resp.Response.Balance != nil:
			realBalance = float64(*resp.Response.Balance)
		default:
			return
		}
		cash := realBalance / 100
		logger.Warning(fmt.Sprintf("Available cash amount: %v", cash))
	}
}
