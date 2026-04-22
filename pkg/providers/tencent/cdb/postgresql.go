package cdb

import (
	"context"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

func (d *Driver) ListPostgreSQL(ctx context.Context) ([]schema.Database, error) {
	list := []schema.Database{}
	d.partialErr = nil
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List PostgreSQL ...")
	}
	var regions []string
	client := d.newClient()
	if d.Region == "all" {
		resp, err := client.DescribePostgresRegions(ctx, d.Region)
		if err != nil {
			logger.Error("List regions failed:", err)
			return list, err
		}
		for _, r := range resp.Response.RegionSet {
			if strings.EqualFold(derefString(r.RegionState), "AVAILABLE") {
				addRegion(&regions, derefString(r.Region))
			}
		}
	} else {
		addRegion(&regions, normalizedRegion(d.Region))
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
		response, probeErr := client.DescribePostgresInstances(ctx, probeRegion)
		if probeErr != nil {
			if api.IsAccessDenied(probeErr) {
				return list, probeErr
			}
			seedErrs[probeRegion] = probeErr
			tracker.Update(probeRegion, 0)
			trackerUsed = true
		} else {
			for _, instance := range response.Response.DBInstanceSet {
				_db := schema.Database{
					InstanceId:    derefString(instance.DBInstanceID),
					Engine:        derefString(instance.DBEngine),
					EngineVersion: derefString(instance.DBInstanceVersion),
					Region:        derefString(instance.Region),
				}
			netLoop:
				for _, info := range instance.DBInstanceNetInfo {
					netType := derefString(info.NetType)
					status := derefString(info.Status)
					if strings.EqualFold(netType, "public") && strings.EqualFold(status, "opened") {
						_db.Address = formatAddressUint64(info.Address, info.Port)
						break netLoop
					}
					if _db.Address == "" && strings.EqualFold(status, "opened") {
						_db.Address = formatAddressUint64(info.IP, info.Port)
					}
				}
				list = append(list, _db)
			}
			tracker.Update(probeRegion, len(response.Response.DBInstanceSet))
			trackerUsed = true
		}
		regions = regions[1:]
	}
	if len(regions) == 0 {
		d.partialErr = regionrun.Wrap(seedErrs)
		return list, nil
	}

	trackerUsed = true
	got, regionErrs := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, r string) ([]schema.Database, error) {
		var regionList []schema.Database
		response, err := client.DescribePostgresInstances(ctx, r)
		if err != nil {
			return regionList, err
		}
		for _, instance := range response.Response.DBInstanceSet {
			_db := schema.Database{
				InstanceId:    derefString(instance.DBInstanceID),
				Engine:        derefString(instance.DBEngine),
				EngineVersion: derefString(instance.DBInstanceVersion),
				Region:        derefString(instance.Region),
			}
		netLoop:
			for _, info := range instance.DBInstanceNetInfo {
				netType := derefString(info.NetType)
				status := derefString(info.Status)
				if strings.EqualFold(netType, "public") && strings.EqualFold(status, "opened") {
					_db.Address = formatAddressUint64(info.Address, info.Port)
					break netLoop
				}
				if _db.Address == "" && strings.EqualFold(status, "opened") {
					_db.Address = formatAddressUint64(info.IP, info.Port)
				}
			}
			regionList = append(regionList, _db)
		}
		return regionList, nil
	})
	list = append(list, got...)
	d.partialErr = regionrun.Wrap(mergeRegionErrors(seedErrs, regionErrs))
	return list, nil
}
