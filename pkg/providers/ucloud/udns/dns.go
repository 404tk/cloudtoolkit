package udns

import (
	"context"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	ucloudauth "github.com/404tk/cloudtoolkit/pkg/providers/ucloud/auth"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

const pageSize = 100

type Driver struct {
	Credential ucloudauth.Credential
	Client     *api.Client
	ProjectID  string
	Regions    []string
}

func (d *Driver) GetDomains(ctx context.Context) ([]schema.Domain, error) {
	list := []schema.Domain{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List DNS ...")
	}
	if len(d.Regions) == 0 {
		return list, nil
	}

	tracker := processbar.NewRegionTracker()
	defer tracker.Finish()

	got, regionErrs := regionrun.ForEach(ctx, d.Regions, 0, tracker, func(ctx context.Context, region string) ([]schema.Domain, error) {
		return d.listRegion(ctx, region)
	})
	list = append(list, mergeDomains(got)...)
	return list, regionrun.Wrap(regionErrs)
}

func (d *Driver) listRegion(ctx context.Context, region string) ([]schema.Domain, error) {
	zones, err := paginate.Fetch[api.ZoneInfo, int](ctx, func(ctx context.Context, offset int) (paginate.Page[api.ZoneInfo, int], error) {
		var resp api.DescribeUDNSZoneResponse
		err := d.client().Do(ctx, api.Request{
			Action: "DescribeUDNSZone",
			Region: region,
			Params: map[string]any{
				"Limit":  pageSize,
				"Offset": offset,
			},
		}, &resp)
		if err != nil {
			return paginate.Page[api.ZoneInfo, int]{}, err
		}

		next := offset + len(resp.DNSZoneInfos)
		done := len(resp.DNSZoneInfos) == 0 || len(resp.DNSZoneInfos) < pageSize
		if resp.TotalCount > 0 {
			done = next >= resp.TotalCount
		}
		return paginate.Page[api.ZoneInfo, int]{
			Items: resp.DNSZoneInfos,
			Next:  next,
			Done:  done,
		}, nil
	})
	if err != nil {
		return nil, err
	}

	list := make([]schema.Domain, 0, len(zones))
	for _, zone := range zones {
		zoneID := strings.TrimSpace(zone.DNSZoneID)
		domainName := strings.TrimSpace(zone.DNSZoneName)
		if zoneID == "" || domainName == "" {
			continue
		}

		records, err := d.listRecords(ctx, region, zoneID)
		if err != nil {
			return list, fmt.Errorf("zone %s: %w", domainName, err)
		}
		list = append(list, schema.Domain{
			DomainName: domainName,
			Records:    records,
		})
	}
	return list, nil
}

func (d *Driver) listRecords(ctx context.Context, region, zoneID string) ([]schema.Record, error) {
	records, err := paginate.Fetch[api.RecordInfo, int](ctx, func(ctx context.Context, offset int) (paginate.Page[api.RecordInfo, int], error) {
		var resp api.DescribeUDNSRecordResponse
		err := d.client().Do(ctx, api.Request{
			Action: "DescribeUDNSRecord",
			Region: region,
			Params: map[string]any{
				"DNSZoneId": zoneID,
				"Limit":     pageSize,
				"Offset":    offset,
			},
		}, &resp)
		if err != nil {
			return paginate.Page[api.RecordInfo, int]{}, err
		}

		next := offset + len(resp.RecordInfos)
		done := len(resp.RecordInfos) == 0 || len(resp.RecordInfos) < pageSize
		if resp.TotalCount > 0 {
			done = next >= resp.TotalCount
		}
		return paginate.Page[api.RecordInfo, int]{
			Items: resp.RecordInfos,
			Next:  next,
			Done:  done,
		}, nil
	})
	if err != nil {
		return nil, err
	}

	out := make([]schema.Record, 0, len(records))
	for _, record := range records {
		rr := strings.TrimSpace(record.Name)
		if rr == "" {
			rr = "@"
		}

		if len(record.ValueSet) == 0 {
			out = append(out, schema.Record{
				RR:     rr,
				Type:   strings.TrimSpace(record.Type),
				Status: "DISABLE",
			})
			continue
		}

		for _, value := range record.ValueSet {
			out = append(out, schema.Record{
				RR:     rr,
				Type:   strings.TrimSpace(record.Type),
				Value:  strings.TrimSpace(value.Data),
				Status: recordStatus(value.IsEnabled),
			})
		}
	}
	return out, nil
}

func (d *Driver) client() *api.Client {
	if d.Client != nil {
		return d.Client
	}
	return api.NewClient(d.Credential, api.WithProjectID(d.ProjectID))
}

func mergeDomains(items []schema.Domain) []schema.Domain {
	indexByName := make(map[string]int, len(items))
	recordSeen := make(map[string]map[string]struct{}, len(items))
	out := make([]schema.Domain, 0, len(items))

	for _, item := range items {
		name := strings.TrimSpace(item.DomainName)
		if name == "" {
			continue
		}

		idx, ok := indexByName[name]
		if !ok {
			indexByName[name] = len(out)
			recordSeen[name] = make(map[string]struct{})
			out = append(out, schema.Domain{DomainName: name})
			idx = len(out) - 1
		}

		for _, record := range item.Records {
			key := fmt.Sprintf("%s|%s|%s|%s", record.RR, record.Type, record.Value, record.Status)
			if _, exists := recordSeen[name][key]; exists {
				continue
			}
			recordSeen[name][key] = struct{}{}
			out[idx].Records = append(out[idx].Records, record)
		}
	}

	return out
}

func recordStatus(enabled int) string {
	if enabled == 0 {
		return "DISABLE"
	}
	return "ENABLE"
}
