package cloudtrail

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

const (
	defaultLookupRegion  = "us-east-1"
	defaultMaxResults    = 50
	maxPages             = 20
)

// Driver wraps the AWS CloudTrail `LookupEvents` action. CloudTrail records
// management-plane operations across the account; the validation flow uses
// `dump` to pull the recent slice and cross-reference what a CSPM detector
// observed. CloudTrail is read-only — `whitelist` returns a clear error.
type Driver struct {
	Client        *api.Client
	Region        string
	DefaultRegion string
}

// DumpEvents returns recent CloudTrail entries. `args` accepts an optional
// `<startUnix>:<endUnix>` time window; pass "" to use the CloudTrail default
// 90-day lookback.
func (d *Driver) DumpEvents(ctx context.Context, args string) ([]schema.Event, error) {
	if d == nil || d.Client == nil {
		return nil, errors.New("aws cloudtrail: nil api client")
	}
	startTime, endTime, err := parseTimeWindow(args)
	if err != nil {
		return nil, err
	}
	region := d.requestRegion()
	out := make([]schema.Event, 0)
	nextToken := ""
	for page := 0; page < maxPages; page++ {
		resp, err := d.Client.CloudTrailLookupEvents(ctx, region, startTime, endTime, defaultMaxResults, nextToken)
		if err != nil {
			return out, err
		}
		for _, ev := range resp.Events {
			out = append(out, schema.Event{
				Id:        ev.EventID,
				Name:      ev.EventName,
				Affected:  firstResourceName(ev.Resources),
				API:       ev.EventName,
				Status:    "成功", // CloudTrail surfaces failures via the embedded CloudTrailEvent JSON; default to success here
				SourceIp:  "",   // SourceIPAddress lives in the embedded JSON blob; left empty in the summary view
				AccessKey: ev.AccessKeyID,
				Time:      formatEventTime(ev.EventTime),
			})
		}
		if resp.NextToken == "" || len(resp.Events) == 0 {
			break
		}
		nextToken = resp.NextToken
	}
	return out, nil
}

// HandleEvents is intentionally not supported — CloudTrail is read-only.
func (d *Driver) HandleEvents(ctx context.Context, _ string) (schema.EventActionResult, error) {
	return schema.EventActionResult{}, errors.New("aws cloudtrail: whitelist action is not supported (CloudTrail is read-only)")
}

func (d *Driver) requestRegion() string {
	region := strings.TrimSpace(d.Region)
	if region == "" || region == "all" {
		region = strings.TrimSpace(d.DefaultRegion)
	}
	if region == "" {
		return defaultLookupRegion
	}
	return region
}

func parseTimeWindow(args string) (int64, int64, error) {
	args = strings.TrimSpace(args)
	if args == "" || args == "all" {
		return 0, 0, nil
	}
	parts := strings.SplitN(args, ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected `<startUnix>:<endUnix>` time window, got %q", args)
	}
	start, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid start unix: %w", err)
	}
	end, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid end unix: %w", err)
	}
	if end < start {
		return 0, 0, fmt.Errorf("end unix %d must be >= start unix %d", end, start)
	}
	return start, end, nil
}

func firstResourceName(resources []api.CloudTrailResource) string {
	for _, r := range resources {
		if name := strings.TrimSpace(r.ResourceName); name != "" {
			return name
		}
	}
	return ""
}

func formatEventTime(unix float64) string {
	if unix <= 0 {
		return ""
	}
	sec := int64(unix)
	return time.Unix(sec, 0).UTC().Format(time.RFC3339)
}
