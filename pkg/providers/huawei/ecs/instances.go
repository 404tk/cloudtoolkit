package ecs

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"

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
	tracker := processbar.NewRegionTracker()
	defer tracker.Finish()
	var regionErrs []string
	var errMu sync.Mutex
	got, _ := regionrun.ForEach(ctx, d.Regions, 0, tracker, func(ctx context.Context, r string) ([]schema.Host, error) {
		projectID, err := api.ResolveProjectID(ctx, d.client(), d.DomainID, r)
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
				Region:     r,
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
					Region:      r,
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
		if err != nil {
			errMu.Lock()
			regionErrs = append(regionErrs, fmt.Sprintf("%s: %s", r, err))
			errMu.Unlock()
		}
		return items, nil
	})
	list = append(list, got...)

	if len(regionErrs) > 0 {
		return list, fmt.Errorf("%s", strings.Join(regionErrs, "; "))
	}
	return list, nil
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
