package cdb

import (
	"context"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

func (d *Driver) ListSQLServer(ctx context.Context) ([]schema.Database, error) {
	list := []schema.Database{}
	d.partialErr = nil
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List SQLServer ...")
	}
	var regions []string
	client := d.newClient()
	if d.Region == "all" {
		resp, err := client.DescribeSQLServerRegions(ctx, d.Region)
		if err != nil {
			logger.Error("list regions failed.")
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

	tracker := processbar.NewRegionTracker()
	defer tracker.Finish()
	got, regionErrs := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, r string) ([]schema.Database, error) {
		var regionList []schema.Database
		response, err := client.DescribeSQLServerInstances(ctx, r)
		if err != nil {
			return regionList, err
		}
		for _, instance := range response.Response.DBInstances {
			_db := schema.Database{
				InstanceId:    derefString(instance.InstanceID),
				Engine:        derefString(instance.VersionName),
				EngineVersion: derefString(instance.Version),
				Region:        derefString(instance.Region),
			}
			if derefString(instance.DNSPodDomain) != "" {
				_db.Address = formatAddressInt64(instance.DNSPodDomain, instance.TgwWanVPort)
			} else {
				_db.Address = formatAddressInt64(instance.Vip, instance.Vport)
			}
			regionList = append(regionList, _db)
		}
		return regionList, nil
	})
	list = append(list, got...)
	d.partialErr = regionrun.Wrap(regionErrs)
	return list, nil
}
