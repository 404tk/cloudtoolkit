package ecs

import (
	"context"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

type Driver struct {
	Cred          aliauth.Credential
	Region        string
	clientOptions []api.Option
	pollInterval  time.Duration
	maxPolls      int
	sleep         func(time.Duration)
	partialErr    error
}

// GetResource returns all the resources in the store for a provider.
func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := []schema.Host{}
	d.partialErr = nil
	select {
	case <-ctx.Done():
		return list, nil
	default:
	}
	logger.Info("List ECS instances...")
	defer func() { SetCacheHostList(list) }()
	client := d.newClient()
	var regions []string
	if d.Region == "all" {
		resp, err := client.DescribeECSRegions(ctx, api.DefaultRegion)
		if err != nil {
			logger.Error("Describe regions failed.")
			return list, err
		}
		for _, r := range resp.Regions.Region {
			regions = append(regions, r.RegionID)
		}
	} else {
		regions = append(regions, api.NormalizeRegion(d.Region))
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
	got, regionErrs := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, r string) ([]schema.Host, error) {
		return d.listRegion(ctx, client, r)
	})
	list = append(list, got...)
	d.partialErr = regionrun.Wrap(mergeRegionErrors(seedErrs, regionErrs))
	return list, nil
}

func (d *Driver) PartialError() error {
	return d.partialErr
}

func (d *Driver) listRegion(ctx context.Context, client *api.Client, region string) ([]schema.Host, error) {
	return paginate.Fetch(ctx, func(ctx context.Context, page int) (paginate.Page[schema.Host, int], error) {
		if page == 0 {
			page = 1
		}
		response, err := client.DescribeECSInstances(ctx, region, page, 100)
		if err != nil {
			return paginate.Page[schema.Host, int]{}, err
		}
		items := mapInstances(region, response.Instances.Instance)
		return paginate.Page[schema.Host, int]{
			Items: items,
			Next:  page + 1,
			Done:  isLastPage(page, response.PageSize, response.TotalCount, len(response.Instances.Instance)),
		}, nil
	})
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

func mapInstances(region string, instances []api.ECSInstance) []schema.Host {
	items := make([]schema.Host, 0, len(instances))
	for _, instance := range instances {
		ipv4 := resolvePublicIPv4(instance)
		privateIPv4 := resolvePrivateIPv4(instance)
		items = append(items, schema.Host{
			HostName:    instance.HostName,
			ID:          instance.InstanceID,
			PublicIPv4:  ipv4,
			PrivateIpv4: privateIPv4,
			OSType:      instance.OSType,
			Public:      ipv4 != "",
			Region:      region,
		})
	}
	return items
}

func resolvePublicIPv4(instance api.ECSInstance) string {
	if len(instance.PublicIP.IPAddress) > 0 {
		return instance.PublicIP.IPAddress[0]
	}
	return instance.EIPAddress.IPAddress
}

func resolvePrivateIPv4(instance api.ECSInstance) string {
	if len(instance.NetworkInterfaces.NetworkInterface) > 0 {
		first := instance.NetworkInterfaces.NetworkInterface[0]
		if len(first.PrivateIPSets.PrivateIPSet) > 0 {
			return first.PrivateIPSets.PrivateIPSet[0].PrivateIPAddress
		}
	}
	for _, netif := range instance.NetworkInterfaces.NetworkInterface {
		if netif.PrimaryIPAddress != "" {
			return netif.PrimaryIPAddress
		}
	}
	return ""
}

func isLastPage(page, pageSize, totalCount, items int) bool {
	if items == 0 {
		return true
	}
	if pageSize <= 0 {
		pageSize = items
	}
	if totalCount <= 0 {
		return items < pageSize
	}
	return page*pageSize >= totalCount
}
