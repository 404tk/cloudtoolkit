package rds

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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
	Cred      auth.Credential
	Regions   []string
	DomainID  string
	Client    *api.Client
	projectID map[string]string
}

func (d *Driver) client() *api.Client {
	if d.Client == nil {
		d.Client = api.NewClient(d.Cred)
	}
	return d.Client
}

func (d *Driver) GetDatabases(ctx context.Context) ([]schema.Database, error) {
	list := []schema.Database{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List RDS instances ...")
	}

	regions := append([]string(nil), d.Regions...)
	regionErrs := make([]string, 0, len(regions))
	tracker := processbar.NewRegionTracker()
	trackerUsed := false
	defer func() {
		if trackerUsed {
			tracker.Finish()
		}
	}()
	if len(regions) > 0 {
		probeRegion := regions[0]
		probeItems, probeErr := d.listRegionDatabases(ctx, probeRegion)
		if probeErr != nil {
			switch {
			case api.IsProjectNotFound(probeErr):
				logger.Warning("Skip RDS region", probeRegion, ":", probeErr.Error())
				tracker.Update(probeRegion, 0)
				trackerUsed = true
			case api.IsAccessDenied(probeErr):
				return list, probeErr
			default:
				regionErrs = append(regionErrs, fmt.Sprintf("%s: %s", probeRegion, probeErr))
				tracker.Update(probeRegion, 0)
				trackerUsed = true
			}
		} else {
			list = append(list, probeItems...)
			tracker.Update(probeRegion, len(probeItems))
			trackerUsed = true
		}
		regions = regions[1:]
	}
	if len(regions) == 0 {
		if len(regionErrs) > 0 {
			return list, fmt.Errorf("%s", strings.Join(regionErrs, "; "))
		}
		return list, nil
	}

	trackerUsed = true
	got, runErrs := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, region string) ([]schema.Database, error) {
		instances, err := d.listRegionDatabases(ctx, region)
		if err != nil {
			if api.IsProjectNotFound(err) {
				logger.Warning("Skip RDS region", region, ":", err.Error())
				return nil, nil
			}
			return nil, err
		}
		return instances, nil
	})
	list = append(list, got...)
	for _, region := range regions {
		err := runErrs[region]
		if err == nil {
			continue
		}
		regionErrs = append(regionErrs, fmt.Sprintf("%s: %s", region, err))
	}

	if len(regionErrs) > 0 {
		return list, fmt.Errorf("%s", strings.Join(regionErrs, "; "))
	}
	return list, nil
}

func (d *Driver) listRegionDatabases(ctx context.Context, region string) ([]schema.Database, error) {
	projectID, err := d.resolveProjectID(ctx, region)
	if err != nil {
		return nil, err
	}

	const limit = int32(100)
	return paginate.Fetch(ctx, func(ctx context.Context, offset int32) (paginate.Page[schema.Database, int32], error) {
		query := url.Values{}
		query.Set("limit", strconv.Itoa(int(limit)))
		query.Set("offset", strconv.Itoa(int(offset)))

		var resp api.ListRDSInstancesResponse
		if err := d.client().DoJSON(ctx, api.Request{
			Service:    "rds",
			Region:     region,
			Intl:       d.Cred.Intl,
			Method:     http.MethodGet,
			Path:       fmt.Sprintf("/v3/%s/instances", projectID),
			Query:      query,
			Idempotent: true,
		}, &resp); err != nil {
			return paginate.Page[schema.Database, int32]{}, err
		}

		list := make([]schema.Database, 0, len(resp.Instances))
		for _, instance := range resp.Instances {
			item := schema.Database{
				InstanceId: instance.ID,
				Region:     strings.TrimSpace(instance.Region),
			}
			if item.Region == "" {
				item.Region = region
			}
			if instance.Datastore != nil {
				item.Engine = strings.TrimSpace(instance.Datastore.Type)
				item.EngineVersion = strings.TrimSpace(instance.Datastore.Version)
			}
			if item.Address = joinAddresses(instance.PublicIPs, instance.Port); item.Address == "" {
				item.Address = joinAddresses(instance.PrivateIPs, instance.Port)
			}
			list = append(list, item)
		}

		done := len(resp.Instances) == 0 ||
			(resp.TotalCount != nil && offset+int32(len(resp.Instances)) >= *resp.TotalCount) ||
			(resp.TotalCount == nil && int32(len(resp.Instances)) < limit)
		return paginate.Page[schema.Database, int32]{
			Items: list,
			Next:  offset + limit,
			Done:  done,
		}, nil
	})
}

func (d *Driver) resolveProjectID(ctx context.Context, region string) (string, error) {
	if d.projectID == nil {
		d.projectID = make(map[string]string)
	}
	if cached := strings.TrimSpace(d.projectID[region]); cached != "" {
		return cached, nil
	}
	projectID, err := api.ResolveProjectID(ctx, d.client(), d.DomainID, region)
	if err != nil {
		return "", err
	}
	d.projectID[region] = projectID
	return projectID, nil
}

func joinAddresses(ips []string, port int32) string {
	if len(ips) == 0 {
		return ""
	}
	addrs := make([]string, 0, len(ips))
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			continue
		}
		addrs = append(addrs, fmt.Sprintf("%s:%d", ip, port))
	}
	return strings.Join(addrs, "\n")
}
