package logs

import (
	"context"
	"errors"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

// Driver enumerates AWS CloudWatch Logs log groups across one or all regions
// and surfaces them as the cloudlist `log` asset. Per-region failures are
// captured via PartialError so a single denied region does not abort the
// remaining cloudlist work.
type Driver struct {
	Client        *api.Client
	Region        string
	DefaultRegion string
	// AvailableRegions is the optional caller-supplied region set used when
	// `Region == "all"`. When empty the driver falls back to a small built-in
	// list of commercial-partition regions; a real probe via DescribeRegions
	// happens upstream in the EC2 driver and is not duplicated here.
	AvailableRegions []string
	partialErr       error
}

var errNilAPIClient = errors.New("aws cloudwatch logs: nil api client")

// fallbackRegions covers the commercial AWS partition. Operators in govcloud
// or AWS China should set Region explicitly; "all" with no AvailableRegions
// keeps the surface predictable.
var fallbackRegions = []string{
	"us-east-1", "us-east-2", "us-west-2", "eu-west-1", "ap-southeast-1",
}

const describeLogGroupsLimit = 50

// GetLogs returns one schema.Log per log group, surfaced as
// `<region>/<logGroupName>` with the storage size in description.
func (d *Driver) GetLogs(ctx context.Context) ([]schema.Log, error) {
	list := []schema.Log{}
	d.partialErr = nil
	if d == nil || d.Client == nil {
		return list, errNilAPIClient
	}
	logger.Info("List CloudWatch log groups ...")

	regions := d.resolveRegions()
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
		d.partialErr = regionrun.Wrap(seedErrs)
		return list, nil
	}

	trackerUsed = true
	got, regionErrs := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, region string) ([]schema.Log, error) {
		return d.listRegion(ctx, region)
	})
	list = append(list, got...)
	d.partialErr = regionrun.Wrap(mergeRegionErrors(seedErrs, regionErrs))
	return list, nil
}

// PartialError returns the aggregated per-region errors collected during the
// last GetLogs call (nil when every region succeeded).
func (d *Driver) PartialError() error {
	return d.partialErr
}

func (d *Driver) listRegion(ctx context.Context, region string) ([]schema.Log, error) {
	items, err := paginate.Fetch[schema.Log, string](ctx, func(ctx context.Context, token string) (paginate.Page[schema.Log, string], error) {
		resp, err := d.Client.CloudWatchLogsDescribeLogGroups(ctx, region, describeLogGroupsLimit, token)
		if err != nil {
			return paginate.Page[schema.Log, string]{}, err
		}
		out := make([]schema.Log, 0, len(resp.LogGroups))
		for _, g := range resp.LogGroups {
			out = append(out, schema.Log{
				ProjectName:    g.LogGroupName,
				Region:         region,
				Description:    g.Arn,
				LastModifyTime: g.CreationTimeFormatted(),
			})
		}
		return paginate.Page[schema.Log, string]{
			Items: out,
			Next:  resp.NextToken,
			Done:  resp.NextToken == "",
		}, nil
	})
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (d *Driver) resolveRegions() []string {
	if d.Region != "" && d.Region != "all" {
		return []string{d.Region}
	}
	if len(d.AvailableRegions) > 0 {
		return append([]string(nil), d.AvailableRegions...)
	}
	return append([]string(nil), fallbackRegions...)
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
