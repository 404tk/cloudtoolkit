package bss

import (
	"context"
	"fmt"
	"net/http"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Cred   auth.Credential
	Client *api.Client
}

func (d *Driver) client() *api.Client {
	if d.Client == nil {
		d.Client = api.NewClient(d.Cred)
	}
	return d.Client
}

func (d *Driver) QueryAccountBalance(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	var resp api.ShowCustomerAccountBalancesResponse
	if err := d.client().DoJSON(ctx, api.Request{
		Service:    "bss",
		Intl:       d.Cred.Intl,
		Method:     http.MethodGet,
		Path:       "/v2/accounts/customer-accounts/balances",
		Idempotent: true,
	}, &resp); err != nil {
		return
	}

	for _, account := range resp.AccountBalances {
		if account.AccountType == 1 {
			logger.Warning(fmt.Sprintf("Available cash amount: %v", account.Amount))
			return
		}
	}
}
