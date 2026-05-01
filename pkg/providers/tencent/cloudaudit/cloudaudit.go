package cloudaudit

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// Driver wraps the tencent CloudAudit `LookUpEvents` action so the
// validation flow can dump recent operation log entries the same way it does
// for alibaba's SAS suspicious events.
type Driver struct {
	Credential    auth.Credential
	clientOptions []api.Option
}

func (d *Driver) SetClientOptions(opts ...api.Option) {
	d.clientOptions = append([]api.Option(nil), opts...)
}

func (d *Driver) newClient() *api.Client {
	return api.NewClient(d.Credential, d.clientOptions...)
}

const (
	defaultLookupRegion  = "ap-guangzhou"
	defaultMaxResults    = 50
	maxResultsPerRequest = 50
	maxPages             = 20
)

// DumpEvents returns recent CloudAudit events. `args` is interpreted as an
// optional `<startUnix>:<endUnix>` time window; pass "" to use the
// CloudAudit default lookback (last 7 days).
func (d *Driver) DumpEvents(ctx context.Context, args string) ([]schema.Event, error) {
	startTime, endTime, err := parseTimeWindow(args)
	if err != nil {
		return nil, err
	}
	client := d.newClient()
	out := make([]schema.Event, 0)
	nextToken := ""
	for page := 0; page < maxPages; page++ {
		resp, err := client.LookUpEvents(ctx, defaultLookupRegion, startTime, endTime, defaultMaxResults, nextToken)
		if err != nil {
			return out, err
		}
		for _, ev := range resp.Response.Events {
			out = append(out, schema.Event{
				Id:        derefString(ev.EventID),
				Name:      derefString(ev.EventNameCn),
				Affected:  derefString(ev.ResourceName),
				API:       derefString(ev.EventName),
				Status:    statusLabel(derefUint64(ev.Status)),
				SourceIp:  derefString(ev.SourceIPAddress),
				AccessKey: derefString(ev.SecretID),
				Time:      derefString(ev.EventTime),
			})
		}
		if resp.Response.ListOver != nil && *resp.Response.ListOver {
			break
		}
		if resp.Response.NextToken == nil || *resp.Response.NextToken == "" {
			break
		}
		nextToken = *resp.Response.NextToken
	}
	return out, nil
}

// HandleEvents is intentionally not implemented — Tencent CloudAudit is a
// read-only audit log and has no equivalent of alibaba SAS's "advance mark
// mis-info" whitelisting flow. We surface a clear error so the REPL doesn't
// silently no-op.
func (d *Driver) HandleEvents(ctx context.Context, _ string) (schema.EventActionResult, error) {
	return schema.EventActionResult{}, errors.New("tencent cloudaudit: whitelist action is not supported (CloudAudit is read-only)")
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

func statusLabel(code uint64) string {
	switch code {
	case 0:
		return "成功"
	case 1:
		return "失败"
	case 2:
		return "部分失败"
	}
	return ""
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefUint64(p *uint64) uint64 {
	if p == nil {
		return 0
	}
	return *p
}
