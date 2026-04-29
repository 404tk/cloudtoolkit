package vm

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const vmPageSize = 100

var knownJDCloudVMRegions = []string{
	"cn-north-1",
	"cn-east-2",
	"cn-east-1",
	"cn-south-1",
}

type pageCursor struct {
	PageNumber int
	Seen       int
}

type Driver struct {
	Client *api.Client
	Region string
}

var (
	cacheHostList []schema.Host
	hostCacheMu   sync.RWMutex
)

// SetCacheHostList stores the hosts enumerated by GetResource so the console
// shell layer can later resolve a host's region + osType without re-listing.
func SetCacheHostList(hosts []schema.Host) {
	hostCacheMu.Lock()
	defer hostCacheMu.Unlock()
	cacheHostList = append([]schema.Host(nil), hosts...)
}

// GetCacheHostList returns a snapshot of the last enumerated host list.
func GetCacheHostList() []schema.Host {
	hostCacheMu.RLock()
	defer hostCacheMu.RUnlock()
	return append([]schema.Host(nil), cacheHostList...)
}

func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := []schema.Host{}
	if ctx == nil {
		ctx = context.Background()
	}
	logger.Info("List VM instances ...")
	if d.Client == nil {
		return list, errors.New("jdcloud vm: nil api client")
	}

	var err error
	regions := d.requestRegions()
	if len(regions) == 1 {
		list, err = d.listRegion(ctx, regions[0])
	} else {
		got, regionErrs := regionrun.ForEach(ctx, regions, 0, nil, func(ctx context.Context, region string) ([]schema.Host, error) {
			return d.listRegion(ctx, region)
		})
		list = append(list, got...)
		err = regionrun.Wrap(regionErrs)
	}

	if len(list) > 0 || err == nil {
		SetCacheHostList(list)
	}
	if err != nil {
		logger.Error("List instances failed.")
	}
	return list, err
}

func (d *Driver) listRegion(ctx context.Context, region string) ([]schema.Host, error) {
	return paginate.Fetch[schema.Host, pageCursor](ctx, func(ctx context.Context, cursor pageCursor) (paginate.Page[schema.Host, pageCursor], error) {
		pageNumber := cursor.PageNumber
		if pageNumber <= 0 {
			pageNumber = 1
		}

		query := url.Values{}
		query.Set("pageNumber", strconv.Itoa(pageNumber))
		query.Set("pageSize", strconv.Itoa(vmPageSize))

		var resp api.DescribeInstancesResponse
		err := d.Client.DoJSON(ctx, api.Request{
			Service: "vm",
			Region:  region,
			Method:  "GET",
			Version: "v1",
			Path:    "/regions/" + region + "/instances",
			Query:   query,
		}, &resp)
		if err != nil {
			return paginate.Page[schema.Host, pageCursor]{}, err
		}

		items := make([]schema.Host, 0, len(resp.Result.Instances))
		for _, i := range resp.Result.Instances {
			ipv4 := i.ElasticIPAddress
			items = append(items, schema.Host{
				HostName:    i.Hostname,
				ID:          i.InstanceID,
				State:       i.Status,
				PublicIPv4:  ipv4,
				PrivateIpv4: i.PrivateIPAddress,
				OSType:      i.OSType,
				Public:      ipv4 != "",
				Region:      region,
			})
		}

		total := resp.Result.TotalCount
		nextSeen := cursor.Seen + len(items)
		done := len(items) == 0
		if total > 0 {
			done = done || nextSeen >= total
		} else {
			done = done || len(items) < vmPageSize
		}
		return paginate.Page[schema.Host, pageCursor]{
			Items: items,
			Next: pageCursor{
				PageNumber: pageNumber + 1,
				Seen:       nextSeen,
			},
			Done: done,
		}, nil
	})
}

func (d *Driver) requestRegions() []string {
	if d.normalizedRegion() == "all" {
		return append([]string(nil), knownJDCloudVMRegions...)
	}
	return []string{d.requestRegion()}
}

func (d *Driver) requestRegion() string {
	region := d.normalizedRegion()
	if region == "" || region == "all" {
		return "cn-north-1"
	}
	return region
}

func (d *Driver) normalizedRegion() string {
	region := strings.TrimSpace(d.Region)
	if strings.EqualFold(region, "all") {
		return "all"
	}
	return region
}
