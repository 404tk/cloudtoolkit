package insights

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	azapi "github.com/404tk/cloudtoolkit/pkg/providers/azure/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// Driver wraps the Azure Activity Log REST surface
// (`Microsoft.Insights/eventtypes/management/values`).
type Driver struct {
	Client          *azapi.Client
	SubscriptionIDs []string
}

const (
	defaultLookbackHours = 24 * 7
	maxPages             = 20
)

// DumpEvents returns recent management-plane activity log events. `args` may
// be `<startUnix>:<endUnix>` (eventTimestamp window), "all", or empty for the
// service default 7-day lookback.
func (d *Driver) DumpEvents(ctx context.Context, args string) ([]schema.Event, error) {
	if d == nil || d.Client == nil {
		return nil, errors.New("azure insights: nil api client")
	}
	if len(d.SubscriptionIDs) == 0 || strings.TrimSpace(d.SubscriptionIDs[0]) == "" {
		return nil, errors.New("azure insights: no subscription configured")
	}
	startTS, endTS, err := parseTimeWindow(args)
	if err != nil {
		return nil, err
	}
	out := make([]schema.Event, 0)
	for _, sub := range d.SubscriptionIDs {
		filter := buildFilter(startTS, endTS)
		query := url.Values{}
		query.Set("api-version", azapi.InsightsAPIVersion)
		query.Set("$filter", filter)
		req := azapi.Request{
			Method:     http.MethodGet,
			Path:       fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Insights/eventtypes/management/values", url.PathEscape(sub)),
			Query:      query,
			Idempotent: true,
		}
		pager := azapi.NewPager[azapi.ActivityLogEvent](d.Client, req)
		events, err := pager.All(ctx)
		if err != nil {
			return out, err
		}
		for _, ev := range events {
			out = append(out, schema.Event{
				Id:        ev.EventDataID,
				Name:      ev.OperationName.LocalizedValue,
				Affected:  ev.ResourceID,
				API:       ev.OperationName.Value,
				Status:    ev.Status.Value,
				SourceIp:  ev.HTTPRequest.ClientIPAddress,
				AccessKey: ev.Caller,
				Time:      ev.EventTimestamp,
			})
		}
	}
	return out, nil
}

// HandleEvents is unsupported — Activity Log is read-only.
func (d *Driver) HandleEvents(ctx context.Context, _ string) (schema.EventActionResult, error) {
	return schema.EventActionResult{}, errors.New("azure insights: whitelist action is not supported (Activity Log is read-only)")
}

// buildFilter constructs the OData $filter expression used by the Activity
// Log endpoint. Both bounds are required by the service; default to
// last-7-days when caller omits them.
func buildFilter(start, end int64) string {
	if start <= 0 || end <= 0 {
		now := time.Now().UTC()
		end = now.Unix()
		start = now.Add(-time.Hour * defaultLookbackHours).Unix()
	}
	startISO := time.Unix(start, 0).UTC().Format(time.RFC3339)
	endISO := time.Unix(end, 0).UTC().Format(time.RFC3339)
	return fmt.Sprintf("eventTimestamp ge '%s' and eventTimestamp le '%s'", startISO, endISO)
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
