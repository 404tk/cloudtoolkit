package uact

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	ucloudauth "github.com/404tk/cloudtoolkit/pkg/providers/ucloud/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// Driver wraps UCloud Action Trail (UACT) lookups used by event-check. Replay
// fixtures and focused tests cover the action-log request/response contract
// used by this validation path.
type Driver struct {
	Credential ucloudauth.Credential
	Client     *api.Client
	ProjectID  string
}

func (d *Driver) client() *api.Client {
	if d.Client != nil {
		return d.Client
	}
	return api.NewClient(d.Credential)
}

const (
	defaultPageSize = 50
	maxPages        = 20
)

// DumpEvents returns recent action-log entries. `args` may be a
// `<startUnix>:<endUnix>` time window, "all", or empty.
func (d *Driver) DumpEvents(ctx context.Context, args string) ([]schema.Event, error) {
	startTime, endTime, err := parseTimeWindow(args)
	if err != nil {
		return nil, err
	}
	out := make([]schema.Event, 0)
	nextToken := ""
	for page := 0; page < maxPages; page++ {
		params := map[string]any{
			"Limit": strconv.Itoa(defaultPageSize),
		}
		if d.ProjectID != "" {
			params["ProjectId"] = d.ProjectID
		}
		if startTime > 0 {
			params["StartTime"] = strconv.FormatInt(startTime, 10)
		}
		if endTime > 0 {
			params["EndTime"] = strconv.FormatInt(endTime, 10)
		}
		if nextToken != "" {
			params["NextToken"] = nextToken
		}
		var resp api.DescribeActionLogListResponse
		err := d.client().Do(ctx, api.Request{
			Action:     "DescribeActionLogList",
			Params:     params,
			Idempotent: true,
		}, &resp)
		if err != nil {
			return out, err
		}
		for _, ev := range resp.Events {
			out = append(out, schema.Event{
				Id:        ev.EventID,
				Name:      ev.EventName,
				Affected:  ev.ResourceName,
				API:       ev.EventName,
				Status:    ev.Status,
				SourceIp:  ev.SourceIPAddress,
				AccessKey: ev.AccessKey,
				Time:      ev.EventTime,
			})
		}
		if resp.NextToken == "" {
			break
		}
		nextToken = resp.NextToken
	}
	return out, nil
}

// HandleEvents is unsupported — UCloud Action Trail is read-only.
func (d *Driver) HandleEvents(ctx context.Context, _ string) (schema.EventActionResult, error) {
	return schema.EventActionResult{}, errors.New("ucloud uact: whitelist action is not supported (Action Trail is read-only)")
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
