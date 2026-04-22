package ecs

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

type Driver struct {
	Client       *api.Client
	Region       string
	pollInterval time.Duration
	maxPolls     int
	sleep        func(time.Duration)
}

var errNilAPIClient = errors.New("volcengine ecs: nil api client")

var (
	cacheHostList []schema.Host
	hostCacheMu   sync.RWMutex
)

func SetCacheHostList(hosts []schema.Host) {
	hostCacheMu.Lock()
	defer hostCacheMu.Unlock()
	cacheHostList = hosts
}

func GetCacheHostList() []schema.Host {
	hostCacheMu.RLock()
	defer hostCacheMu.RUnlock()
	return cacheHostList
}

func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := []schema.Host{}
	logger.Info("List ECS instances ...")
	defer func() { SetCacheHostList(list) }()
	client, err := d.requireClient()
	if err != nil {
		return list, err
	}

	regions, err := d.getRegions(ctx, client)
	if err != nil {
		logger.Error("List regions failed.")
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
		return list, regionrun.Wrap(seedErrs)
	}

	trackerUsed = true
	got, regionErrs := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, r string) ([]schema.Host, error) {
		return d.listRegion(ctx, client, r)
	})
	list = append(list, got...)
	return list, regionrun.Wrap(mergeRegionErrors(seedErrs, regionErrs))
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

func (d *Driver) listRegion(ctx context.Context, client *api.Client, region string) ([]schema.Host, error) {
	return paginate.Fetch[schema.Host, string](ctx, func(ctx context.Context, token string) (paginate.Page[schema.Host, string], error) {
		resp, err := client.DescribeInstances(ctx, region, 100, token)
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
				Region:      region,
			})
		}
		done := len(resp.Result.Instances) < 100 || strings.TrimSpace(resp.Result.NextToken) == ""
		return paginate.Page[schema.Host, string]{
			Items: items,
			Next:  resp.Result.NextToken,
			Done:  done,
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

func (d *Driver) requestRegion() string {
	region := strings.TrimSpace(d.Region)
	if region == "" || region == "all" {
		return api.DefaultRegion
	}
	return region
}
