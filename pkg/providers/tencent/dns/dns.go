package dns

import (
	"context"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Credential    auth.Credential
	Region        string
	clientOptions []api.Option
}

func (d *Driver) newClient() *api.Client {
	return api.NewClient(d.Credential, d.clientOptions...)
}

func (d *Driver) SetClientOptions(opts ...api.Option) {
	d.clientOptions = append([]api.Option(nil), opts...)
}

func (d *Driver) GetDomains(ctx context.Context) ([]schema.Domain, error) {
	list := []schema.Domain{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List DNS ...")
	}
	client := d.newClient()
	domains, err := paginate.Fetch(ctx, func(ctx context.Context, offset int64) (paginate.Page[api.DomainListItem, int64], error) {
		response, err := client.DescribeDomainList(ctx, d.Region, offset, 3000)
		if err != nil {
			logger.Error("DescribeDomainList failed.")
			return paginate.Page[api.DomainListItem, int64]{}, err
		}
		total := derefUint64(response.Response.DomainCountInfo.DomainTotal)
		items := response.Response.DomainList
		return paginate.Page[api.DomainListItem, int64]{
			Items: items,
			Next:  offset + int64(len(items)),
			Done:  doneByTotal(offset, int64(len(items)), int64(total), 3000),
		}, nil
	})
	if err != nil {
		return list, err
	}
	for _, domain := range domains {
		if !enabledDomain(domain) {
			continue
		}
		name := derefString(domain.Name)
		if name == "" {
			continue
		}
		_domain := schema.Domain{DomainName: name}
		records, err := paginate.Fetch(ctx, func(ctx context.Context, offset uint64) (paginate.Page[api.RecordListItem, uint64], error) {
			resp, err := client.DescribeRecordList(ctx, d.Region, name, offset, 3000)
			if err != nil {
				return paginate.Page[api.RecordListItem, uint64]{}, err
			}
			total := derefUint64(resp.Response.RecordCountInfo.TotalCount)
			items := resp.Response.RecordList
			return paginate.Page[api.RecordListItem, uint64]{
				Items: items,
				Next:  offset + uint64(len(items)),
				Done:  doneByTotal(offset, uint64(len(items)), total, uint64(3000)),
			}, nil
		})
		if err != nil {
			return list, err
		}
		for _, record := range records {
			_domain.Records = append(_domain.Records, schema.Record{
				RR:     derefString(record.Name),
				Type:   derefString(record.Type),
				Value:  derefString(record.Value),
				Status: derefString(record.Status),
			})
		}
		list = append(list, _domain)
	}
	return list, nil
}

func enabledDomain(domain api.DomainListItem) bool {
	return strings.EqualFold(derefString(domain.Status), "ENABLE") &&
		!strings.EqualFold(derefString(domain.DNSStatus), "DNSERROR")
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func derefUint64(v *uint64) uint64 {
	if v == nil {
		return 0
	}
	return *v
}

type integer interface {
	~int | ~int64 | ~uint | ~uint64
}

func doneByTotal[T integer](offset, count, total, limit T) bool {
	if count == 0 {
		return true
	}
	if total <= 0 {
		return count < limit
	}
	return offset+count >= total
}
