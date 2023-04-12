package bss

import (
	"errors"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/global"
	bss "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/bss/v2"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/bss/v2/model"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/bss/v2/region"
)

func QueryBalance(auth basic.Credentials) (float64, error) {
	_auth := global.NewCredentialsBuilder().
		WithAk(auth.AK).
		WithSk(auth.SK).
		Build()
	client := bss.NewBssClient(
		bss.BssClientBuilder().
			WithRegion(region.ValueOf("cn-north-1")).
			WithCredential(_auth).
			Build())

	request := &model.ShowCustomerAccountBalancesRequest{}
	response, err := client.ShowCustomerAccountBalances(request)
	if err != nil {
		return 0, err
	}
	for _, account := range *response.AccountBalances {
		if account.AccountType == 1 {
			return account.Amount, nil
		}
	}
	return 0, errors.New(response.String())
}
