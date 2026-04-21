package ecs

import (
	"context"
	"errors"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

type Driver struct {
	Client *api.Client
	Region string
}

var errNilAPIClient = errors.New("volcengine ecs: nil api client")

func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := []schema.Host{}
	logger.Info("List ECS instances ...")
	client, err := d.requireClient()
	if err != nil {
		return list, err
	}

	regions, err := d.getRegions(ctx, client)
	if err != nil {
		logger.Error("List regions failed.")
		return list, err
	}
	tracker := processbar.NewRegionTracker()
	defer tracker.Finish()
	got, regionErrs := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, r string) ([]schema.Host, error) {
		return paginate.Fetch[schema.Host, string](ctx, func(ctx context.Context, token string) (paginate.Page[schema.Host, string], error) {
			resp, err := client.DescribeInstances(ctx, r, 100, token)
			if err != nil {
				return paginate.Page[schema.Host, string]{}, err
			}
			items := make([]schema.Host, 0, len(resp.Result.Instances))
			for _, i := range resp.Result.Instances {
				ipv4 := i.EipAddress.IPAddress
				var privateIPv4 string
				if len(i.NetworkInterfaces) > 0 {
					privateIPv4 = i.NetworkInterfaces[0].PrimaryIPAddress
				}
				items = append(items, schema.Host{
					HostName:    i.Hostname,
					ID:          i.InstanceID,
					State:       i.Status,
					PublicIPv4:  ipv4,
					PrivateIpv4: privateIPv4,
					OSType:      i.OSType,
					Public:      ipv4 != "",
					Region:      r,
				})
			}
			done := len(resp.Result.Instances) < 100 || strings.TrimSpace(resp.Result.NextToken) == ""
			return paginate.Page[schema.Host, string]{
				Items: items,
				Next:  resp.Result.NextToken,
				Done:  done,
			}, nil
		})
	})
	list = append(list, got...)
	return list, regionrun.Wrap(regionErrs)
}

func (d *Driver) requireClient() (*api.Client, error) {
	if d.Client == nil {
		return nil, errNilAPIClient
	}
	return d.Client, nil
}

func (d *Driver) getRegions(ctx context.Context, client *api.Client) ([]string, error) {
	if d.Region != "all" {
		return []string{d.requestRegion()}, nil
	}
	resp, err := client.DescribeRegions(ctx, d.requestRegion(), 100)
	if err != nil {
		return nil, err
	}
	regions := make([]string, 0, len(resp.Result.Regions))
	for _, region := range resp.Result.Regions {
		if id := strings.TrimSpace(region.RegionID); id != "" {
			regions = append(regions, id)
		}
	}
	return regions, nil
}

func (d *Driver) requestRegion() string {
	region := strings.TrimSpace(d.Region)
	if region == "" || region == "all" {
		return api.DefaultRegion
	}
	return region
}
