package dns

import (
	"context"
	"errors"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const pageSize int32 = 100

type Driver struct {
	Client *api.Client
}

var errNilAPIClient = errors.New("volcengine dns: nil api client")

func (d *Driver) GetDomains(ctx context.Context) ([]schema.Domain, error) {
	list := []schema.Domain{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List DNS ...")
	}
	if d.Client == nil {
		return list, errNilAPIClient
	}

	zones, err := paginate.Fetch[api.DNSZone, int32](ctx, func(ctx context.Context, pageNumber int32) (paginate.Page[api.DNSZone, int32], error) {
		if pageNumber == 0 {
			pageNumber = 1
		}
		resp, err := d.Client.ListDNSZones(ctx, pageNumber, pageSize)
		if err != nil {
			logger.Error("List zones failed.")
			return paginate.Page[api.DNSZone, int32]{}, err
		}
		return paginate.Page[api.DNSZone, int32]{
			Items: resp.Zones,
			Next:  pageNumber + 1,
			Done:  pageDone(pageNumber, pageSize, resp.Total, len(resp.Zones)),
		}, nil
	})
	if err != nil {
		return list, err
	}

	for _, zone := range zones {
		name := strings.TrimSpace(zone.ZoneName)
		if name == "" || zone.ZID == 0 {
			continue
		}
		records, err := d.listRecords(ctx, zone.ZID)
		if err != nil {
			logger.Error("List records failed.")
			return list, err
		}
		list = append(list, schema.Domain{
			DomainName: name,
			Records:    records,
		})
	}

	return list, nil
}

func (d *Driver) listRecords(ctx context.Context, zid int64) ([]schema.Record, error) {
	records, err := paginate.Fetch[api.DNSRecord, int32](ctx, func(ctx context.Context, pageNumber int32) (paginate.Page[api.DNSRecord, int32], error) {
		if pageNumber == 0 {
			pageNumber = 1
		}
		resp, err := d.Client.ListDNSRecords(ctx, zid, pageNumber, pageSize)
		if err != nil {
			return paginate.Page[api.DNSRecord, int32]{}, err
		}
		return paginate.Page[api.DNSRecord, int32]{
			Items: resp.Records,
			Next:  pageNumber + 1,
			Done:  pageDone(pageNumber, pageSize, resp.TotalCount, len(resp.Records)),
		}, nil
	})
	if err != nil {
		return nil, err
	}

	list := make([]schema.Record, 0, len(records))
	for _, record := range records {
		list = append(list, schema.Record{
			RR:     firstNonEmpty(strings.TrimSpace(record.Host), strings.TrimSpace(record.FQDN), "@"),
			Type:   strings.TrimSpace(record.Type),
			Value:  strings.TrimSpace(record.Value),
			Status: recordStatus(record.Enable),
		})
	}
	return list, nil
}

func recordStatus(enable *bool) string {
	if enable == nil {
		return ""
	}
	if *enable {
		return "ENABLE"
	}
	return "DISABLE"
}

func pageDone(pageNumber, pageSize, total int32, items int) bool {
	if items == 0 {
		return true
	}
	if total <= 0 {
		return int32(items) < pageSize
	}
	return pageNumber*pageSize >= total
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
