package cdb

import (
	"context"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

func (d *Driver) ListPostgreSQL(ctx context.Context) ([]schema.Database, error) {
	list := []schema.Database{}
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

	tracker := processbar.NewRegionTracker()
	defer tracker.Finish()
	got, _ := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, r string) ([]schema.Database, error) {
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
	return list, nil
}
