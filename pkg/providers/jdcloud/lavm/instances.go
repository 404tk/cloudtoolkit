package lavm

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const lavmPageSize = 100

var knownJDCloudLAVMRegions = []string{
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

func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := []schema.Host{}
	if ctx == nil {
		ctx = context.Background()
	}
	logger.Info("List LAVM instances ...")
	if d.Client == nil {
		return list, errors.New("jdcloud lavm: nil api client")
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
		filteredErrs := filterInvalidRegionErrors(regionErrs)
		if len(filteredErrs) == 0 && len(list) == 0 {
			err = regionrun.Wrap(regionErrs)
		} else {
			err = regionrun.Wrap(filteredErrs)
		}
	}

	if err != nil {
		logger.Error("List LAVM instances failed.")
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
		query.Set("pageSize", strconv.Itoa(lavmPageSize))

		var resp api.DescribeLAVMInstancesResponse
		err := d.Client.DoJSON(ctx, api.Request{
			Service: "lavm",
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
		for _, instance := range resp.Result.Instances {
			hostRegion := strings.TrimSpace(instance.RegionID)
			if hostRegion == "" {
				hostRegion = region
			}
			publicIP := strings.TrimSpace(instance.PublicIPAddress)
			items = append(items, schema.Host{
				HostName:    strings.TrimSpace(instance.InstanceName),
				ID:          strings.TrimSpace(instance.InstanceID),
				State:       hostState(instance),
				PublicIPv4:  publicIP,
				PrivateIpv4: strings.TrimSpace(instance.PrivateIPAddress),
				DNSName:     firstDomain(instance.Domains),
				Public:      publicIP != "",
				Region:      hostRegion,
			})
		}

		total := resp.Result.TotalCount
		nextSeen := cursor.Seen + len(items)
		done := len(items) == 0
		if total > 0 {
			done = done || nextSeen >= total
		} else {
			done = done || len(items) < lavmPageSize
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
		return append([]string(nil), knownJDCloudLAVMRegions...)
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

func hostState(instance api.LAVMInstance) string {
	if state := strings.TrimSpace(instance.Status); state != "" {
		return state
	}
	return strings.TrimSpace(instance.BusinessStatus)
}

func firstDomain(domains []api.LAVMDomain) string {
	for _, domain := range domains {
		if name := strings.TrimSpace(domain.DomainName); name != "" {
			return name
		}
	}
	return ""
}

func filterInvalidRegionErrors(errs map[string]error) map[string]error {
	if len(errs) == 0 {
		return nil
	}
	filtered := make(map[string]error, len(errs))
	for region, err := range errs {
		if api.IsInvalidRegion(err) {
			continue
		}
		filtered[region] = err
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}
