package cts

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

const (
	defaultRegion    = "cn-north-4"
	defaultPageLimit = 200
	maxPages         = 20
)

// Driver wraps Huawei CTS `ListTraces` so event-check can review recent
// management-plane operations from an authorized environment. CTS is read-only:
// `whitelist` returns a clear unsupported error instead of silently no-oping.
type Driver struct {
	Cred      auth.Credential
	Regions   []string
	DomainID  string
	Client    *api.Client
	projectID map[string]string
}

func (d *Driver) client() *api.Client {
	if d.Client == nil {
		d.Client = api.NewClient(d.Cred)
	}
	return d.Client
}

// DumpEvents queries CTS management traces and optionally filters the result by
// source IP when `sourceFilter` is not empty and not "all".
func (d *Driver) DumpEvents(ctx context.Context, sourceFilter string) ([]schema.Event, error) {
	if d == nil {
		return nil, errors.New("huawei cts: nil driver")
	}

	regions := d.resolveRegions()
	out := make([]schema.Event, 0)
	regionErrs := make([]string, 0)
	for _, region := range regions {
		events, err := d.listRegionEvents(ctx, region, sourceFilter)
		if err != nil {
			switch {
			case api.IsProjectNotFound(err):
				continue
			case api.IsAccessDenied(err):
				return out, err
			default:
				regionErrs = append(regionErrs, fmt.Sprintf("%s: %v", region, err))
				continue
			}
		}
		out = append(out, events...)
	}
	if len(regionErrs) > 0 {
		return out, errors.New(strings.Join(regionErrs, "; "))
	}
	return out, nil
}

// HandleEvents is intentionally not supported — Huawei CTS is an audit log,
// not a configurable detection/whitelist surface.
func (d *Driver) HandleEvents(ctx context.Context, _ string) (schema.EventActionResult, error) {
	return schema.EventActionResult{}, errors.New("huawei cts: whitelist action is not supported (CTS is read-only)")
}

func (d *Driver) listRegionEvents(ctx context.Context, region, sourceFilter string) ([]schema.Event, error) {
	projectID, err := d.resolveProjectID(ctx, region)
	if err != nil {
		return nil, err
	}

	sourceFilter = strings.TrimSpace(sourceFilter)
	query := url.Values{}
	query.Set("trace_type", "system")
	query.Set("tracker_name", "system")
	query.Set("limit", strconv.Itoa(defaultPageLimit))

	out := make([]schema.Event, 0)
	for page := 0; page < maxPages; page++ {
		var resp api.ListTracesResponse
		if err := d.client().DoJSON(ctx, api.Request{
			Service:    "cts",
			Region:     region,
			Intl:       d.Cred.Intl,
			Method:     http.MethodGet,
			Path:       fmt.Sprintf("/v3/%s/traces", projectID),
			Query:      query,
			Idempotent: true,
		}, &resp); err != nil {
			return out, err
		}
		for _, trace := range resp.Traces {
			if !matchSourceIP(trace.SourceIP, sourceFilter) {
				continue
			}
			out = append(out, schema.Event{
				Id:        strings.TrimSpace(trace.TraceID),
				Name:      firstNonEmpty(trace.TraceName, trace.OperationID),
				Affected:  firstNonEmpty(trace.ResourceName, trace.ResourceID),
				API:       firstNonEmpty(trace.OperationID, trace.TraceName),
				Status:    statusLabel(trace.Code, trace.TraceRating),
				SourceIp:  strings.TrimSpace(trace.SourceIP),
				AccessKey: strings.TrimSpace(trace.User.AccessKeyID),
				Time:      formatUnixMillis(trace.Time),
			})
		}
		next := strings.TrimSpace(resp.MetaData.Marker)
		if next == "" || len(resp.Traces) == 0 {
			break
		}
		query.Set("next", next)
	}
	return out, nil
}

func (d *Driver) resolveProjectID(ctx context.Context, region string) (string, error) {
	if d.projectID == nil {
		d.projectID = make(map[string]string)
	}
	if cached := strings.TrimSpace(d.projectID[region]); cached != "" {
		return cached, nil
	}
	projectID, err := api.ResolveProjectID(ctx, d.client(), d.DomainID, region)
	if err != nil {
		return "", err
	}
	d.projectID[region] = projectID
	return projectID, nil
}

func (d *Driver) resolveRegions() []string {
	if len(d.Regions) > 0 {
		regions := make([]string, 0, len(d.Regions))
		seen := make(map[string]struct{}, len(d.Regions))
		for _, region := range d.Regions {
			region = strings.TrimSpace(region)
			if region == "" {
				continue
			}
			if _, ok := seen[region]; ok {
				continue
			}
			seen[region] = struct{}{}
			regions = append(regions, region)
		}
		if len(regions) > 0 {
			return regions
		}
	}

	region := strings.TrimSpace(d.Cred.Region)
	if region == "" || region == "all" {
		region = defaultRegion
	}
	return []string{region}
}

func matchSourceIP(value, filter string) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" || strings.EqualFold(filter, "all") {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(value), filter)
}

func statusLabel(code, rating string) string {
	code = strings.TrimSpace(code)
	if len(code) > 0 {
		switch code[0] {
		case '2', '3':
			return "成功"
		default:
			return "失败"
		}
	}
	switch strings.ToLower(strings.TrimSpace(rating)) {
	case "normal":
		return "正常"
	case "warning":
		return "告警"
	case "incident":
		return "事件"
	default:
		return ""
	}
}

func formatUnixMillis(ms int64) string {
	if ms <= 0 {
		return ""
	}
	return time.UnixMilli(ms).UTC().Format(time.RFC3339)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" && trimmed != "-" {
			return trimmed
		}
	}
	return ""
}
