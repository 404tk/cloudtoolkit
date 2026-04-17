package ecs

import (
	"context"
	"math"

	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
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

func (d *Driver) NewClient() (*ecs.Client, error) {
	region := d.Region
	if region == "all" || region == "" {
		region = "cn-hangzhou"
	}
	return ecs.NewClientWithOptions(region, sdk.NewConfig(), d.Cred)
}

// GetResource returns all the resources in the store for a provider.
func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := []schema.Host{}
	logger.Info("List ECS instances...")
	defer func() { SetCacheHostList(list) }()
	client, err := d.NewClient()
	if err != nil {
		return list, err
	}
	// check permission
	vpc_client, err := vpc.NewClientWithOptions("cn-hangzhou", sdk.NewConfig(), d.Cred)
	if err != nil {
		return list, err
	}
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
	tracker := processbar.NewRegionTracker()
	defer tracker.Finish()
	got, _ := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, r string) ([]schema.Host, error) {
		return paginate.Fetch(ctx, func(ctx context.Context, page int) (paginate.Page[schema.Host, int], error) {
			if page == 0 {
				page = 1
			}
			request := ecs.CreateDescribeInstancesRequest()
			request.PageSize = requests.NewInteger(100)
			request.PageNumber = requests.NewInteger(page)
			request.RegionId = r
			response, err := client.DescribeInstances(request)
			if err != nil {
				return paginate.Page[schema.Host, int]{}, err
			}
			items := make([]schema.Host, 0, len(response.Instances.Instance))
			for _, instance := range response.Instances.Instance {
				var ipv4, privateIPv4 string
				if len(instance.PublicIpAddress.IpAddress) > 0 {
					ipv4 = instance.PublicIpAddress.IpAddress[0]
				}
				if len(instance.NetworkInterfaces.NetworkInterface) > 0 && len(instance.NetworkInterfaces.NetworkInterface[0].PrivateIpSets.PrivateIpSet) > 0 {
					privateIPv4 = instance.NetworkInterfaces.NetworkInterface[0].PrivateIpSets.PrivateIpSet[0].PrivateIpAddress
				}
				if privateIPv4 == "" {
					for _, net := range instance.NetworkInterfaces.NetworkInterface {
						if net.PrimaryIpAddress != "" {
							privateIPv4 = net.PrimaryIpAddress
						}
					}
				}
				if ipv4 == "" {
					ipv4 = instance.EipAddress.IpAddress
				}
				items = append(items, schema.Host{
					HostName:    instance.HostName,
					ID:          instance.InstanceId,
					PublicIPv4:  ipv4,
					PrivateIpv4: privateIPv4,
					OSType:      instance.OSType,
					Public:      ipv4 != "",
					Region:      r,
				})
			}
			pageCount := int(math.Ceil(float64(response.TotalCount) / 100))
			return paginate.Page[schema.Host, int]{
				Items: items,
				Next:  page + 1,
				Done:  page >= pageCount,
			}, nil
		})
	})
	list = append(list, got...)

	return list, nil
}
