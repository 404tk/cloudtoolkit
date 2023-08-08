package billing

import (
	"context"
	"log"

	billing "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/billing/v20180709"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

type Driver struct {
	Cred   *common.Credential
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
		region = "ap-guangzhou"
	}
	cpf := profile.NewClientProfile()
	// cpf.HttpProfile.Endpoint = "billing.tencentcloudapi.com"
	client, _ := billing.NewClient(d.Cred, region, cpf)
	req_billing := billing.NewDescribeAccountBalanceRequest()
	resp_billing, err := client.DescribeAccountBalance(req_billing)
	if err == nil {
		cash := *resp_billing.Response.RealBalance / 100
		log.Printf("[+] Available cash amount: %v\n", cash)
	}
}
