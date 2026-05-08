package actiontrail

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// Driver wraps the JDCloud AuditTrail audit log lookup used by event-check.
// Replay fixtures and focused tests cover the request/response contract used
// by this validation path.
type Driver struct {
	Client    *api.Client
	Region    string
	AccessKey string
}

const (
	defaultEventLimit = 20
	defaultPageSize   = defaultEventLimit
	maxPages          = 1
)

// DumpEvents returns recent AuditTrail events. `args` is interpreted as
// `<startUnix>:<endUnix>` (or "all" / empty for service default lookback).
func (d *Driver) DumpEvents(ctx context.Context, args string) ([]schema.Event, error) {
	if d == nil || d.Client == nil {
		return nil, errors.New("jdcloud audittrail: nil api client")
	}
	startTime, endTime, err := parseTimeWindow(args)
	if err != nil {
		return nil, err
	}
	out := make([]schema.Event, 0, defaultEventLimit)
	seen := int64(0)
	lookupAttributes := accessKeyLookupAttributes(d.AccessKey)
	for page := 1; page <= maxPages; page++ {
		resp, err := d.Client.DescribeActionTrailEvents(ctx, d.Region, startTime, endTime, page, defaultPageSize, lookupAttributes)
		if err != nil {
			return out, err
		}
		for _, ev := range resp.Result.Events {
			if len(out) >= defaultEventLimit {
				break
			}
			out = append(out, schema.Event{
				// Id:   ev.EventID,
				Name: ev.EventName,
				// API:      ev.EventName,
				Status:   eventStatus(ev.ErrorCode, ev.ErrorMessage),
				SourceIp: ev.IP,
				// AccessKey: ev.AccessKeyID,
				Time: formatEventTime(ev.EventTime),
			})
		}
		seen += int64(len(resp.Result.Events))
		if len(resp.Result.Events) == 0 {
			break
		}
		if len(out) >= defaultEventLimit {
			break
		}
		if resp.Result.TotalNumber > 0 && seen >= resp.Result.TotalNumber {
			break
		}
		if len(resp.Result.Events) < defaultPageSize {
			break
		}
	}
	return out, nil
}

// HandleEvents is unsupported: JDCloud AuditTrail is read-only with no
// whitelist concept.
func (d *Driver) HandleEvents(ctx context.Context, _ string) (schema.EventActionResult, error) {
	return schema.EventActionResult{}, errors.New("jdcloud audittrail: whitelist action is not supported (AuditTrail is read-only)")
}

func eventStatus(errorCode, errorMessage string) string {
	if strings.TrimSpace(errorCode) != "" || strings.TrimSpace(errorMessage) != "" {
		return "Failed"
	}
	return "Success"
}

func formatEventTime(value api.ActionTrailTimestamp) string {
	ts := int64(value)
	if ts <= 0 {
		return ""
	}
	if ts > 9999999999 {
		return time.UnixMilli(ts).UTC().Format(time.RFC3339)
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}

func accessKeyLookupAttributes(accessKey string) string {
	accessKey = strings.TrimSpace(accessKey)
	if accessKey == "" {
		return ""
	}
	raw, err := json.Marshal(map[string]string{"accessKeyId": accessKey})
	if err != nil {
		return ""
	}
	return string(raw)
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
