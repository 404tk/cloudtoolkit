package lighthouse

import (
	"context"
	"fmt"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	lighthouse "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/lighthouse/v20200324"
)

type InstanceProvider struct {
	Credential *common.Credential
	Cpf        *profile.ClientProfile
	Region     string
}

// GetResource returns all the resources in the store for a provider.
func (d *InstanceProvider) GetResource(ctx context.Context) ([]*schema.Host, error) {
	list := schema.NewResources().Hosts
	var regions []string
	if d.Region == "" {
		client, err := lighthouse.NewClient(d.Credential, d.Region, d.Cpf)
		if err != nil {
			return list, err
		}
		req := lighthouse.NewDescribeRegionsRequest()
		resp, err := client.DescribeRegions(req)
		for _, r := range resp.Response.RegionSet {
			regions = append(regions, *r.Region)
		}
	} else {
		regions = append(regions, d.Region)
	}
	for _, r := range regions {
		client, err := lighthouse.NewClient(d.Credential, r, d.Cpf)
		if err != nil {
			return list, err
		}

		request := lighthouse.NewDescribeInstancesRequest()
		response, err := client.DescribeInstances(request)
		if err != nil {
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
			_host := &schema.Host{
				PublicIPv4:  ipv4,
				PrivateIpv4: privateIPv4,
				Public:      ipv4 != "",
			}
			list = append(list, _host)
		}
		progress := fmt.Sprintf("Inquiring %s regionId,number of discovered hosts: %d", r, len(response.Response.InstanceSet))
		log.Println(progress)
	}
	return list, nil
}
