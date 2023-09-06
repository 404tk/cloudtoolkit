package bss

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/global"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/as/v1/region"
	bss "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/bss/v2"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/bss/v2/model"
)

type Driver struct {
	Cred   basic.Credentials
	Region string
}

func (d *Driver) QueryAccountBalance(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
	}
	_auth := global.NewCredentialsBuilder().
		WithAk(d.Cred.AK).
		WithSk(d.Cred.SK).
		Build()
	r := d.Region
	if r == "all" {
		r = "cn-north-1"
	}
	client := bss.NewBssClient(
		bss.BssClientBuilder().
			WithRegion(region.ValueOf(r)).
			WithCredential(_auth).
			Build())

	request := &model.ShowCustomerAccountBalancesRequest{}
	response, err := client.ShowCustomerAccountBalances(request)
	if err != nil {
		return
	}
	for _, account := range *response.AccountBalances {
		if account.AccountType == 1 {
			logger.Warning(fmt.Sprintf("Available cash amount: %v", account.Amount))
			return
		}
	}
}
