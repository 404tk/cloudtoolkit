package ecs

import (
	"context"
	"fmt"
	"math"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
)

type Driver struct {
	Cred   *credentials.StsTokenCredential
	Region string
}

// GetResource returns all the resources in the store for a provider.
func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := schema.NewResources().Hosts
	logger.Info("Start enumerating ECS ...")
	region := d.Region
	if region == "all" {
		region = "cn-hangzhou"
	}
	client, err := ecs.NewClientWithOptions(region, sdk.NewConfig(), d.Cred)
	if err != nil {
		return list, err
	}
	// check permission
	vpc_client, _ := vpc.NewClientWithOptions("cn-hangzhou", sdk.NewConfig(), d.Cred)
	req_vpc := vpc.CreateDescribeVpcsRequest()
	_, err = vpc_client.DescribeVpcs(req_vpc)
	if err != nil {
		logger.Error("Describe vpcs failed.")
		return list, err
	}
	var regions []string
	if d.Region == "all" {
		req := ecs.CreateDescribeRegionsRequest()
		resp, err := client.DescribeRegions(req)
		if err != nil {
			logger.Error("Describe regions failed.")
			return list, err
		}
		for _, r := range resp.Regions.Region {
			regions = append(regions, r.RegionId)
		}
	} else {
		regions = append(regions, d.Region)
	}
	flag := false
	prevLength := 0
	count := 0
	for _, r := range regions {
		page := 1
		for {
			request := ecs.CreateDescribeInstancesRequest()
			request.PageSize = requests.NewInteger(100)
			request.PageNumber = requests.NewInteger(page)
			request.RegionId = r
			// Getting a list of instances
			response, err := client.DescribeInstances(request)
			if err != nil {
				break
			}
			pageCount := int(math.Ceil(float64(response.TotalCount) / 100))
			for _, instance := range response.Instances.Instance {
				// Getting Host Information
				var ipv4, privateIPv4 string
				if len(instance.PublicIpAddress.IpAddress) > 0 {
					ipv4 = instance.PublicIpAddress.IpAddress[0]
				}
				if len(instance.NetworkInterfaces.NetworkInterface) > 0 && len(instance.NetworkInterfaces.NetworkInterface[0].PrivateIpSets.PrivateIpSet) > 0 {
					privateIPv4 = instance.NetworkInterfaces.NetworkInterface[0].PrivateIpSets.PrivateIpSet[0].PrivateIpAddress
				}
				if privateIPv4 == "" {
					// Get the primary and private IP addresses from the network adapter configuration
					for _, net := range instance.NetworkInterfaces.NetworkInterface {
						if net.PrimaryIpAddress != "" {
							privateIPv4 = net.PrimaryIpAddress
						}
					}
				}
				if ipv4 == "" {
					// Get the public IP address from the Eip
					ipv4 = instance.EipAddress.IpAddress
				}

				_host := schema.Host{
					HostName:    instance.HostName,
					PublicIPv4:  ipv4,
					PrivateIpv4: privateIPv4,
					Public:      ipv4 != "",
					Region:      r,
				}
				list = append(list, _host)
			}
			if page == pageCount || pageCount == 0 {
				break
			}
			page++
		}
		select {
		case <-ctx.Done():
			goto done
		default:
			prevLength, flag = processbar.RegionPrint(r, len(list)-count, prevLength, flag)
			count = len(list)
		}
	}
done:
	if !flag {
		fmt.Printf("\n\033[F\033[K")
	}

	return list, nil
}
