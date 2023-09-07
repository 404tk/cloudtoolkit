package lighthouse

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	lighthouse "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/lighthouse/v20200324"
)

type Driver struct {
	Credential *common.Credential
	Region     string
}

func (d *Driver) NewClient() (*lighthouse.Client, error) {
	cpf := profile.NewClientProfile()
	region := d.Region
	if region == "all" || region == "" {
		region = "ap-guangzhou"
	}
	return lighthouse.NewClient(d.Credential, region, cpf)
}

// GetResource returns all the resources in the store for a provider.
func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := schema.NewResources().Hosts
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("Start enumerating Lighthouse ...")
	}
	var regions []string
	if d.Region == "all" {
		client, _ := d.NewClient()
		req := lighthouse.NewDescribeRegionsRequest()
		resp, err := client.DescribeRegions(req)
		if err != nil {
			logger.Error("Enumerate Lighthouse failed.")
			return list, err
		}
		for _, r := range resp.Response.RegionSet {
			regions = append(regions, *r.Region)
		}
	} else {
		regions = append(regions, d.Region)
	}

	flag := false
	prevLength := 0
	for _, r := range regions {
		d.Region = r
		client, _ := d.NewClient()
		request := lighthouse.NewDescribeInstancesRequest()
		response, err := client.DescribeInstances(request)
		if err != nil {
			logger.Error("Enumerate Lighthouse failed.")
			return list, err
		}

		for _, instance := range response.Response.InstanceSet {
			var ipv4, privateIPv4 string
			if len(instance.PublicAddresses) > 0 {
				ipv4 = *instance.PublicAddresses[0]
			}
			if len(instance.PrivateAddresses) > 0 {
				privateIPv4 = *instance.PrivateAddresses[0]
			}
			_host := schema.Host{
				HostName:    *instance.InstanceName,
				ID:          *instance.InstanceId,
				PublicIPv4:  ipv4,
				PrivateIpv4: privateIPv4,
				OSType:      *instance.PlatformType, // LINUX_UNIX or WINDOWS
				Public:      ipv4 != "",
				Region:      r,
			}
			list = append(list, _host)
		}
		select {
		case <-ctx.Done():
			goto done
		default:
			prevLength, flag = processbar.RegionPrint(r, len(response.Response.InstanceSet), prevLength, flag)
		}
	}
done:
	if !flag {
		fmt.Printf("\n\033[F\033[K")
	}
	return list, nil
}
