package ec2

import (
	"context"
	"errors"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

type Driver struct {
	Client        *api.Client
	Region        string
	DefaultRegion string
	partialErr    error
}

var errNilAPIClient = errors.New("aws ec2: nil api client")

// GetResource returns all the resources in the store for a provider.
func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := []schema.Host{}
	d.partialErr = nil
	logger.Info("List EC2 instances ...")
	client, err := d.requireClient()
	if err != nil {
		return list, err
	}
	regions, err := d.GetEC2Regions(ctx)
	if err != nil {
		logger.Error("GetEC2Regions failed.")
		return list, err
	}

	seedErrs := map[string]error{}
	tracker := processbar.NewRegionTracker()
	trackerUsed := false
	defer func() {
		if trackerUsed {
			tracker.Finish()
		}
	}()
	if d.Region == "all" && len(regions) > 0 {
		probeRegion := regions[0]
		probeItems, probeErr := d.listRegion(ctx, client, probeRegion)
		if probeErr != nil {
			if api.IsAccessDenied(probeErr) {
				return list, probeErr
			}
			seedErrs[probeRegion] = probeErr
			tracker.Update(probeRegion, 0)
			trackerUsed = true
		} else {
			list = append(list, probeItems...)
			tracker.Update(probeRegion, len(probeItems))
			trackerUsed = true
		}
		regions = regions[1:]
	}
	if len(regions) == 0 {
		d.partialErr = regionrun.Wrap(seedErrs)
		return list, nil
	}

	trackerUsed = true
	got, regionErrs := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, region string) ([]schema.Host, error) {
		return d.listRegion(ctx, client, region)
	})
	list = append(list, got...)
	d.partialErr = regionrun.Wrap(mergeRegionErrors(seedErrs, regionErrs))
	return list, nil
}

func (d *Driver) PartialError() error {
	return d.partialErr
}

func (d *Driver) GetEC2Regions(ctx context.Context) ([]string, error) {
	var regions []string
	if d.Region == "all" {
		client, err := d.requireClient()
		if err != nil {
			return nil, err
		}
		resp, err := client.DescribeRegions(ctx, d.bootstrapRegion())
		if err != nil {
			return nil, err
		}
		for _, region := range resp.Regions {
			if region.Name != "" {
				regions = append(regions, region.Name)
			}
		}
	} else {
		regions = append(regions, d.bootstrapRegion())
	}
	return regions, nil
}

func (d *Driver) requireClient() (*api.Client, error) {
	if d.Client == nil {
		return nil, errNilAPIClient
	}
	return d.Client, nil
}

func (d *Driver) listRegion(ctx context.Context, client *api.Client, region string) ([]schema.Host, error) {
	items, err := paginate.Fetch[schema.Host, string](ctx, func(ctx context.Context, token string) (paginate.Page[schema.Host, string], error) {
		resp, err := client.DescribeInstances(ctx, region, token, 1000)
		if err != nil {
			return paginate.Page[schema.Host, string]{}, err
		}
		hosts := make([]schema.Host, 0, len(resp.Instances))
		for _, instance := range resp.Instances {
			ip4 := instance.PublicIP
			host := schema.Host{
				HostName:    pickHostName(instance.Tags),
				ID:          instance.InstanceID,
				State:       instance.State,
				PublicIPv4:  ip4,
				PrivateIpv4: instance.PrivateIP,
				DNSName:     instance.PublicDNSName,
				Public:      ip4 != "",
				Region:      region,
			}
			hosts = append(hosts, host)
		}
		return paginate.Page[schema.Host, string]{
			Items: hosts,
			Next:  resp.NextToken,
			Done:  resp.NextToken == "",
		}, nil
	})
	if err != nil {
		return nil, err
	}
	return items, nil
}

func mergeRegionErrors(base, extra map[string]error) map[string]error {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	merged := make(map[string]error, len(base)+len(extra))
	for region, err := range base {
		if err != nil {
			merged[region] = err
		}
	}
	for region, err := range extra {
		if err != nil {
			merged[region] = err
		}
	}
	return merged
}

func pickHostName(tags []api.EC2Tag) string {
	var fallback string
	for _, tag := range tags {
		switch tag.Key {
		case "aws:cloudformation:stack-name":
			return tag.Value
		case "Name":
			if fallback == "" {
				fallback = tag.Value
			}
		}
	}
	return fallback
}

func (d *Driver) bootstrapRegion() string {
	region := d.Region
	if region == "" || region == "all" {
		region = d.DefaultRegion
	}
	if region == "" {
		return "us-east-1"
	}
	return region
}
