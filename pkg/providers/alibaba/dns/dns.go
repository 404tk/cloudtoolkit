package dns

import (
	"context"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
)

type Driver struct {
	Cred   *credentials.StsTokenCredential
	Region string
}

func (d *Driver) GetDomains(ctx context.Context) ([]schema.Domain, error) {
	list := schema.NewResources().Domains
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("Start enumerating DNS ...")
	}
	region := d.Region
	if region == "all" {
		region = "cn-hangzhou"
	}
	client, err := alidns.NewClientWithOptions(region, sdk.NewConfig(), d.Cred)
	if err != nil {
		return list, err
	}
	request := alidns.CreateDescribeDomainsRequest()
	request.Scheme = "https"
	response, err := client.DescribeDomains(request)
	if err != nil {
		logger.Error("Describe domains failed.")
		return list, err
	}
	for _, domain := range response.Domains.Domain {
		_domain := schema.Domain{
			DomainName: domain.DomainName,
		}
		req := alidns.CreateDescribeDomainRecordsRequest()
		req.Scheme = "https"
		req.DomainName = domain.DomainName
		resp, err := client.DescribeDomainRecords(req)
		if err != nil {
			return list, err
		}
		for _, record := range resp.DomainRecords.Record {
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
