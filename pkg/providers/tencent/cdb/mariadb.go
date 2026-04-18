package cdb

import (
	"context"

	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

func (d *Driver) ListMariaDB(ctx context.Context) ([]schema.Database, error) {
	list := []schema.Database{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List MariaDB ...")
	}
	var regions []string
	client := d.newClient()
	if d.Region == "all" {
		resp, err := client.DescribeMariaDBSaleInfo(ctx, d.Region)
		if err != nil {
			logger.Error("List regions failed.")
			return list, err
		}
		for _, r := range resp.Response.RegionList {
			addRegion(&regions, derefString(r.Region))
		}
	} else {
		addRegion(&regions, normalizedRegion(d.Region))
	}

	tracker := processbar.NewRegionTracker()
	defer tracker.Finish()
	got, _ := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, r string) ([]schema.Database, error) {
		var regionList []schema.Database
		response, err := client.DescribeMariaDBInstances(ctx, r)
		if err != nil {
			return regionList, err
		}
		for _, instance := range response.Response.Instances {
			_db := schema.Database{
				InstanceId:    derefString(instance.InstanceID),
				Engine:        "MariaDB",
				EngineVersion: derefString(instance.DBVersion),
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
	return list, nil
}
