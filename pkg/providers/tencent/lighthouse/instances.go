package lighthouse

import (
	"context"

	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
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

func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := []schema.Host{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List Lighthouse instances ...")
	}
	var regions []string
	if d.Region == "all" {
		client, err := d.NewClient()
		if err != nil {
			return list, err
		}
		req := lighthouse.NewDescribeRegionsRequest()
		resp, err := client.DescribeRegions(req)
		if err != nil {
			logger.Error("List regions failed.")
			return list, err
		}
		for _, r := range resp.Response.RegionSet {
			regions = append(regions, *r.Region)
		}
	} else {
		regions = append(regions, d.Region)
	}

	tracker := processbar.NewRegionTracker()
	defer tracker.Finish()
	got, _ := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, r string) ([]schema.Host, error) {
		client, err := lighthouse.NewClient(d.Credential, r, profile.NewClientProfile())
		if err != nil {
			return nil, err
		}
		return paginate.Fetch(ctx, func(ctx context.Context, offset int64) (paginate.Page[schema.Host, int64], error) {
			request := lighthouse.NewDescribeInstancesRequest()
			request.Limit = common.Int64Ptr(100)
			request.Offset = common.Int64Ptr(offset)
			response, err := client.DescribeInstances(request)
			if err != nil {
				return paginate.Page[schema.Host, int64]{}, err
			}
			items := make([]schema.Host, 0, len(response.Response.InstanceSet))
			for _, instance := range response.Response.InstanceSet {
				var ipv4, privateIPv4 string
				if len(instance.PublicAddresses) > 0 {
					ipv4 = *instance.PublicAddresses[0]
				}
				if len(instance.PrivateAddresses) > 0 {
					privateIPv4 = *instance.PrivateAddresses[0]
				}
				items = append(items, schema.Host{
					HostName:    *instance.InstanceName,
					ID:          *instance.InstanceId,
					State:       *instance.InstanceState,
					PublicIPv4:  ipv4,
					PrivateIpv4: privateIPv4,
					OSType:      *instance.PlatformType,
					Public:      ipv4 != "",
					Region:      r,
				})
			}
			return paginate.Page[schema.Host, int64]{
				Items: items,
				Next:  offset + 100,
				Done:  len(response.Response.InstanceSet) < 100,
			}, nil
		})
	})
	list = append(list, got...)
	return list, nil
}
