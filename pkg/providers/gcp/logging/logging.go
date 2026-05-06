package logging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// Driver wraps Cloud Logging `entries:list` for the validation flow. The
// payload filter targets cloudaudit.googleapis.com data-access / activity logs
// so the dump surfaces management-plane events relevant to CSPM detection.
type Driver struct {
	Client   *api.Client
	Projects []string
}

const (
	defaultPageSize = 50
	maxPages        = 20
)

// DumpEvents returns recent Cloud Audit log entries. `args` may be a
// `<startUnix>:<endUnix>` time window, "all", or empty (default 7-day
// lookback).
func (d *Driver) DumpEvents(ctx context.Context, args string) ([]schema.Event, error) {
	if d == nil || d.Client == nil {
		return nil, errors.New("gcp logging: nil api client")
	}
	if len(d.Projects) == 0 || strings.TrimSpace(d.Projects[0]) == "" {
		return nil, errors.New("gcp logging: no project configured")
	}
	startTS, endTS, err := parseTimeWindow(args)
	if err != nil {
		return nil, err
	}
	resourceNames := make([]string, 0, len(d.Projects))
	for _, p := range d.Projects {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		resourceNames = append(resourceNames, "projects/"+p)
	}

	out := make([]schema.Event, 0)
	pageToken := ""
	for page := 0; page < maxPages; page++ {
		body := api.ListLogEntriesRequest{
			ResourceNames: resourceNames,
			Filter:        buildFilter(startTS, endTS),
			OrderBy:       "timestamp desc",
			PageSize:      defaultPageSize,
			PageToken:     pageToken,
		}
		raw, err := json.Marshal(body)
		if err != nil {
			return out, err
		}
		var resp api.ListLogEntriesResponse
		err = d.Client.Do(ctx, api.Request{
			Method:  http.MethodPost,
			BaseURL: api.LoggingBaseURL,
			Path:    "/v2/entries:list",
			Body:    raw,
		}, &resp)
		if err != nil {
			return out, err
		}
		for _, entry := range resp.Entries {
			out = append(out, schema.Event{
				Id:        entry.InsertID,
				Name:      entry.ProtoPayload.MethodName,
				Affected:  entry.ProtoPayload.ResourceName,
				API:       entry.ProtoPayload.MethodName,
				Status:    statusLabel(entry.ProtoPayload.Status.Code),
				SourceIp:  entry.ProtoPayload.RequestMeta.CallerIP,
				AccessKey: entry.ProtoPayload.AuthInfo.PrincipalEmail,
				Time:      entry.Timestamp,
			})
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return out, nil
}

// HandleEvents is unsupported — Cloud Audit Logs are read-only.
func (d *Driver) HandleEvents(ctx context.Context, _ string) (schema.EventActionResult, error) {
	return schema.EventActionResult{}, errors.New("gcp logging: whitelist action is not supported (Cloud Audit Logs are read-only)")
}

// buildFilter constructs the entries:list filter expression. We scope to the
// cloudaudit.googleapis.com log family so the dump is dominated by management
// plane events; without it the list is overrun by VM serial console logs etc.
func buildFilter(start, end int64) string {
	var clauses []string
	clauses = append(clauses, `logName:"cloudaudit.googleapis.com"`)
	if start > 0 && end > 0 {
		startISO := time.Unix(start, 0).UTC().Format(time.RFC3339)
		endISO := time.Unix(end, 0).UTC().Format(time.RFC3339)
		clauses = append(clauses,
			fmt.Sprintf(`timestamp>="%s"`, startISO),
			fmt.Sprintf(`timestamp<="%s"`, endISO))
	} else {
		// Default: last 7 days. Cloud Logging accepts unix-epoch math via the
		// `timestamp >=` operator with an absolute RFC3339 value.
		since := time.Now().UTC().Add(-7 * 24 * time.Hour).Format(time.RFC3339)
		clauses = append(clauses, fmt.Sprintf(`timestamp>="%s"`, since))
	}
	return strings.Join(clauses, " AND ")
}

func statusLabel(code int) string {
	if code == 0 {
		return "Success"
	}
	return "Failed(" + strconv.Itoa(code) + ")"
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
