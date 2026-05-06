package rds

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

// AvailableRegions is the optional caller-supplied region set used when the
// driver Region == "all". When empty, the driver falls back to the same
// commercial-partition slice the EC2 driver uses, so cloudlist `all` mode
// surfaces RDS instances without a separate DescribeRegions call.
type databaseAssetDriver struct {
	*Driver
	AvailableRegions []string
}

var defaultRDSRegions = []string{
	"us-east-1", "us-east-2", "us-west-2", "eu-west-1", "ap-southeast-1",
}

// GetDatabases lists RDS instances across one or all regions and surfaces
// them as the cloudlist `database` asset. Per-region failures are recorded
// via PartialError so a denied region does not abort the rest of cloudlist.
func (d *Driver) GetDatabases(ctx context.Context) ([]schema.Database, error) {
	helper := &databaseAssetDriver{Driver: d, AvailableRegions: defaultRDSRegions}
	return helper.collect(ctx)
}

// PartialError returns the aggregated per-region errors collected during the
// last GetDatabases call (nil when every region succeeded).
func (d *Driver) PartialError() error {
	return d.partialErr
}

func (d *databaseAssetDriver) collect(ctx context.Context) ([]schema.Database, error) {
	list := []schema.Database{}
	if d.Driver == nil || d.Driver.Client == nil {
		return list, fmt.Errorf("aws rds: nil api client")
	}
	d.Driver.partialErr = nil
	logger.Info("List RDS instances ...")

	regions := d.resolveRegions()
	seedErrs := map[string]error{}
	tracker := processbar.NewRegionTracker()
	trackerUsed := false
	defer func() {
		if trackerUsed {
			tracker.Finish()
		}
	}()

	if d.Driver.Region == "all" && len(regions) > 0 {
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
		d.Driver.partialErr = regionrun.Wrap(seedErrs)
		return list, nil
	}

	trackerUsed = true
	got, regionErrs := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, region string) ([]schema.Database, error) {
		return d.listRegion(ctx, region)
	})
	list = append(list, got...)
	d.Driver.partialErr = regionrun.Wrap(mergeRegionErrors(seedErrs, regionErrs))
	return list, nil
}

func (d *databaseAssetDriver) listRegion(ctx context.Context, region string) ([]schema.Database, error) {
	items, err := paginate.Fetch[schema.Database, string](ctx, func(ctx context.Context, marker string) (paginate.Page[schema.Database, string], error) {
		resp, err := d.Driver.Client.DescribeDBInstances(ctx, region, marker)
		if err != nil {
			return paginate.Page[schema.Database, string]{}, err
		}
		out := make([]schema.Database, 0, len(resp.DBInstances))
		for _, inst := range resp.DBInstances {
			network := "Private"
			if inst.PubliclyAccessible {
				network = "Public"
			}
			address := inst.Address
			if address != "" && inst.Port > 0 {
				address = fmt.Sprintf("%s:%d", inst.Address, inst.Port)
			}
			out = append(out, schema.Database{
				InstanceId:    inst.DBInstanceIdentifier,
				Engine:        inst.Engine,
				EngineVersion: inst.EngineVersion,
				Region:        region,
				Address:       address,
				NetworkType:   network,
				DBNames:       inst.DBName,
			})
		}
		return paginate.Page[schema.Database, string]{
			Items: out,
			Next:  resp.Marker,
			Done:  resp.Marker == "",
		}, nil
	})
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (d *databaseAssetDriver) resolveRegions() []string {
	if d.Driver.Region != "" && d.Driver.Region != "all" {
		return []string{d.Driver.Region}
	}
	if len(d.AvailableRegions) > 0 {
		return append([]string(nil), d.AvailableRegions...)
	}
	return append([]string(nil), defaultRDSRegions...)
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
