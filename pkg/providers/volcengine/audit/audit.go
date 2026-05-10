package audit

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// Driver wraps Volcengine CloudTrail `LookupEvents` so the validation flow can
// dump operation log entries similarly to other clouds.
type Driver struct {
	Client    *api.Client
	Region    string
	AccessKey string
}

const (
	pageSize = 20
	maxPages = 1
)

// DumpEvents returns recent Audit events. `args` may be a `<startUnix>:<endUnix>`
// time window, "all", or empty (service default lookback).
func (d *Driver) DumpEvents(ctx context.Context, args string) ([]schema.Event, error) {
	if d == nil || d.Client == nil {
		return nil, errors.New("volcengine audit: nil api client")
	}
	startTime, endTime, err := parseTimeWindow(args)
	if err != nil {
		return nil, err
	}
	out := make([]schema.Event, 0)
	nextToken := ""
	for page := 0; page < maxPages; page++ {
		resp, err := d.Client.LookupAuditEvents(ctx, d.Region, startTime, endTime, pageSize, nextToken, d.AccessKey)
		if err != nil {
			return out, err
		}
		for _, ev := range resp.Result.Trails {
			out = append(out, schema.Event{
				// Id:        ev.EventID,
				Name:     ev.EventNameDisplay,
				API:      ev.EventName,
				Status:   auditEventStatus(ev),
				SourceIp: ev.SourceIPAddress,
				// AccessKey: ev.AccessKeyID,
				Time: ev.EventTime,
			})
		}
		if resp.Result.NextToken == "" {
			break
		}
		nextToken = resp.Result.NextToken
	}
	return out, nil
}

// HandleEvents is a no-op for Volcengine CloudTrail: like other vendor audit
// services it is read-only with no whitelist concept. Surface a clear error
// instead of silently succeeding.
func (d *Driver) HandleEvents(ctx context.Context, _ string) (schema.EventActionResult, error) {
	return schema.EventActionResult{}, errors.New("volcengine cloudtrail: whitelist action is not supported (CloudTrail is read-only)")
}

func auditEventStatus(ev api.AuditEvent) string {
	if code := strings.TrimSpace(ev.ErrorCode); code != "" {
		return code
	}
	return "Success"
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
