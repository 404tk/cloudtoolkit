package ecs

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

type Driver struct {
	Cred     auth.Credential
	Regions  []string
	DomainID string
	Client   *api.Client
}

func (d *Driver) client() *api.Client {
	if d.Client == nil {
		d.Client = api.NewClient(d.Cred)
	}
	return d.Client
}

// GetResource returns all the resources in the store for a provider.
func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := []schema.Host{}
	logger.Info("List ECS instances ...")

	seedErrs := map[string]error{}
	tracker := processbar.NewRegionTracker()
	trackerUsed := false
	defer func() {
		if trackerUsed {
			tracker.Finish()
		}
	}()
	regions := append([]string(nil), d.Regions...)
	if len(regions) > 0 {
		probeRegion := regions[0]
		probeItems, probeErr := d.listRegion(ctx, probeRegion)
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
		return d.listRegion(ctx, r)
	})
	list = append(list, got...)
	return list, regionrun.Wrap(mergeRegionErrors(seedErrs, regionErrs))
}

func (d *Driver) listRegion(ctx context.Context, region string) ([]schema.Host, error) {
	projectID, err := api.ResolveProjectID(ctx, d.client(), d.DomainID, region)
	if err != nil {
		return nil, err
	}
	const limit = int32(100)
	items, err := paginate.Fetch(ctx, func(ctx context.Context, page int32) (paginate.Page[schema.Host, int32], error) {
		if page == 0 {
			page = 1
		}
		query := url.Values{}
		query.Set("limit", strconv.Itoa(int(limit)))
		query.Set("offset", strconv.Itoa(int(page)))

		var resp api.ListECSServersDetailsResponse
		err := d.client().DoJSON(ctx, api.Request{
			Service:    "ecs",
			Region:     region,
			Intl:       d.Cred.Intl,
			Method:     http.MethodGet,
			Path:       fmt.Sprintf("/v1/%s/cloudservers/detail", projectID),
			Query:      query,
			Idempotent: true,
		}, &resp)
		if err != nil {
			return paginate.Page[schema.Host, int32]{}, err
		}
		items := make([]schema.Host, 0, len(resp.Servers))
		for _, instance := range resp.Servers {
			ipv4, privateIPv4 := mapHostIPs(instance.Addresses)
			items = append(items, schema.Host{
				State:       instance.Status,
				HostName:    instance.Name,
				PublicIPv4:  ipv4,
				PrivateIpv4: privateIPv4,
				Public:      ipv4 != "",
				Region:      region,
			})
		}
		done := len(resp.Servers) == 0 ||
			(resp.Count > 0 && page*limit >= resp.Count) ||
			(resp.Count == 0 && int32(len(resp.Servers)) < limit)
		return paginate.Page[schema.Host, int32]{
			Items: items,
			Next:  page + 1,
			Done:  done,
		}, nil
	})
	return items, err
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

func mapHostIPs(addresses map[string][]api.ECSServerAddress) (string, string) {
	if len(addresses) == 0 {
		return "", ""
	}

	keys := make([]string, 0, len(addresses))
	for key := range addresses {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var publicIPv4, privateIPv4 string
	for _, key := range keys {
		for _, addr := range addresses[key] {
			switch strings.ToLower(strings.TrimSpace(addr.OSEXTIPStype)) {
			case "floating":
				if publicIPv4 == "" {
					publicIPv4 = strings.TrimSpace(addr.Addr)
				}
			case "fixed":
				if privateIPv4 == "" {
					privateIPv4 = strings.TrimSpace(addr.Addr)
				}
			}
		}
	}
	return publicIPv4, privateIPv4
}
