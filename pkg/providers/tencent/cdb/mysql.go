package cdb

import (
	"context"

	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

func (d *Driver) ListMySQL(ctx context.Context) ([]schema.Database, error) {
	list := []schema.Database{}
	d.partialErr = nil
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List MySQL ...")
	}
	var regions []string
	client := d.newClient()
	if d.Region == "all" {
		resp, err := client.DescribeCDBZoneConfig(ctx, d.Region)
		if err != nil {
			logger.Error("List regions failed.")
			return list, err
		}
		for _, r := range resp.Response.DataResult.Regions {
			addRegion(&regions, derefString(r.Region))
		}
	} else {
		addRegion(&regions, normalizedRegion(d.Region))
	}

	tracker := processbar.NewRegionTracker()
	defer tracker.Finish()
	got, regionErrs := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, r string) ([]schema.Database, error) {
		var regionList []schema.Database
		response, err := client.DescribeCDBInstances(ctx, r)
		if err != nil {
			if unsupportedRegion(err) {
				return regionList, nil
			}
			return regionList, err
		}
		for _, instance := range response.Response.Items {
			_db := schema.Database{
				InstanceId:    derefString(instance.InstanceID),
				Engine:        "MySQL",
				EngineVersion: derefString(instance.EngineVersion),
				Region:        derefString(instance.Region),
			}
			if derefInt64(instance.WanStatus) == 1 {
				_db.Address = formatAddressInt64(instance.WanDomain, instance.WanPort)
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
