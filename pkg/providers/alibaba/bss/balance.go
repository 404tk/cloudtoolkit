package bss

import (
	"context"

	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/bssopenapi"
)

type Driver struct {
	Cred   *credentials.StsTokenCredential
	Region string
}

func (d *Driver) QueryAccountBalance(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
	}
	region := d.Region
	if region == "all" {
		region = "cn-hangzhou"
	}
	bssclient, _ := bssopenapi.NewClientWithOptions(region, sdk.NewConfig(), d.Cred)
	req_bss := bssopenapi.CreateQueryAccountBalanceRequest()
	req_bss.Scheme = "https"
	resp, err := bssclient.QueryAccountBalance(req_bss)
	if err == nil {
		if resp.Data.AvailableCashAmount != "" {
			logger.Warning("Available cash amount:", resp.Data.AvailableCashAmount)
		}
	}
}
