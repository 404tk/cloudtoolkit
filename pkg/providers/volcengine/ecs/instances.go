package ecs

import (
	"context"

	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
	"github.com/volcengine/volcengine-go-sdk/service/ecs"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
)

type Driver struct {
	Conf   *volcengine.Config
	Region string
}

func (d *Driver) NewClient(region string) (*ecs.ECS, error) {
	if region == "all" || region == "" {
		region = "cn-beijing"
	}
	sess, err := session.NewSession(d.Conf.Copy().WithRegion(region))
	if err != nil {
		return nil, err
	}
	return ecs.New(sess), nil
}

func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := []schema.Host{}
	logger.Info("List ECS instances ...")
	svc, err := d.NewClient(d.Region)
	if err != nil {
		return list, err
	}

	var regions []string
	if d.Region == "all" {
		input := &ecs.DescribeRegionsInput{MaxResults: volcengine.Int32(100)}
		resp, err := svc.DescribeRegions(input)
		if err != nil {
			logger.Error("List regions failed.")
			return list, err
		}
		for _, r := range resp.Regions {
			regions = append(regions, *r.RegionId)
		}
	} else {
		regions = append(regions, d.Region)
	}
	tracker := processbar.NewRegionTracker()
	defer tracker.Finish()
	got, _ := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, r string) ([]schema.Host, error) {
		svc, err := d.NewClient(r)
		if err != nil {
			return nil, err
		}
		return paginate.Fetch(ctx, func(ctx context.Context, token *string) (paginate.Page[schema.Host, *string], error) {
			if token == nil {
				token = volcengine.String("")
			}
			resp, err := svc.DescribeInstances(&ecs.DescribeInstancesInput{
				MaxResults: volcengine.Int32(100),
				NextToken:  token,
			})
			if err != nil {
				return paginate.Page[schema.Host, *string]{}, err
			}
			items := make([]schema.Host, 0, len(resp.Instances))
			for _, i := range resp.Instances {
				ipv4 := volcengine.StringValue(i.EipAddress.IpAddress)
				var privateIPv4 string
				if len(i.NetworkInterfaces) > 0 {
					privateIPv4 = volcengine.StringValue(i.NetworkInterfaces[0].PrimaryIpAddress)
				}
				items = append(items, schema.Host{
					HostName:    volcengine.StringValue(i.Hostname),
					ID:          volcengine.StringValue(i.InstanceId),
					State:       volcengine.StringValue(i.Status),
					PublicIPv4:  ipv4,
					PrivateIpv4: privateIPv4,
					OSType:      volcengine.StringValue(i.OsType),
					Public:      ipv4 != "",
					Region:      r,
				})
			}
			done := len(resp.Instances) < 100 || volcengine.StringValue(resp.NextToken) == ""
			return paginate.Page[schema.Host, *string]{
				Items: items,
				Next:  resp.NextToken,
				Done:  done,
			}, nil
		})
	})
	list = append(list, got...)
	return list, nil
}
