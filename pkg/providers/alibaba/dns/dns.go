package dns

import (
	"context"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Cred          aliauth.Credential
	Region        string
	clientOptions []api.Option
}

func (d *Driver) newClient() *api.Client {
	return api.NewClient(d.Cred, d.clientOptions...)
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
		logger.Info("List domains ...")
	}
	client := d.newClient()
	region := api.NormalizeRegion(d.Region)
	domains, err := paginate.Fetch(ctx, func(ctx context.Context, page int) (paginate.Page[api.DomainSummary, int], error) {
		if page == 0 {
			page = 1
		}
		response, err := client.DescribeDomains(ctx, region, page, 100)
		if err != nil {
			logger.Error("Describe domains failed.")
			return paginate.Page[api.DomainSummary, int]{}, err
		}
		return paginate.Page[api.DomainSummary, int]{
			Items: response.Domains.Domain,
			Next:  page + 1,
			Done:  isLastPage(page, response.PageSize, response.TotalCount, len(response.Domains.Domain)),
		}, nil
	})
	if err != nil {
		return list, err
	}
	for _, domain := range domains {
		select {
		case <-ctx.Done():
			return list, nil
		default:
		}
		_domain := schema.Domain{
			DomainName: domain.DomainName,
		}
		records, err := paginate.Fetch(ctx, func(ctx context.Context, page int) (paginate.Page[api.DomainRecord, int], error) {
			if page == 0 {
				page = 1
			}
			resp, err := client.DescribeDomainRecords(ctx, region, domain.DomainName, page, 100)
			if err != nil {
				return paginate.Page[api.DomainRecord, int]{}, err
			}
			return paginate.Page[api.DomainRecord, int]{
				Items: resp.DomainRecords.Record,
				Next:  page + 1,
				Done:  isLastPage(page, resp.PageSize, resp.TotalCount, len(resp.DomainRecords.Record)),
			}, nil
		})
		if err != nil {
			return list, err
		}
		for _, record := range records {
			_domain.Records = append(_domain.Records, schema.Record{
				RR:     record.RR,
				Type:   record.Type,
				Value:  record.Value,
				Status: record.Status,
			})
		}
		list = append(list, _domain)
	}

	return list, nil
}

func isLastPage(page, pageSize, totalCount, items int) bool {
	if items == 0 {
		return true
	}
	if pageSize <= 0 {
		pageSize = items
	}
	if totalCount <= 0 {
		return items < pageSize
	}
	return page*pageSize >= totalCount
}
