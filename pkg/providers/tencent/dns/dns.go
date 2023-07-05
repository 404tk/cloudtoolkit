package dns

import (
	"context"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
)

type Driver struct {
	Credential *common.Credential
}

func (d *Driver) GetDomains(ctx context.Context) ([]schema.Domain, error) {
	list := schema.NewResources().Domains
	select {
	case <-ctx.Done():
		return list, nil
	default:
		log.Println("[*] Start enumerating DNS ...")
	}
	cpf := profile.NewClientProfile()
	//cpf.HttpProfile.Endpoint = "dnspod.tencentcloudapi.com"
	client, _ := dnspod.NewClient(d.Credential, "", cpf)
	request := dnspod.NewDescribeDomainListRequest()
	response, err := client.DescribeDomainList(request)
	if err != nil {
		log.Println("[-] Enumerate CAM failed.")
		return list, err
	}
	for _, domain := range response.Response.DomainList {
		if *domain.Status == "ENABLE" && *domain.DNSStatus != "DNSERROR" {
			_domain := schema.Domain{DomainName: *domain.Name}
			req := dnspod.NewDescribeRecordListRequest()
			req.Domain = domain.Name
			resp, err := client.DescribeRecordList(req)
			if err != nil {
				return list, err
			}
			for _, record := range resp.Response.RecordList {
				_domain.Records = append(_domain.Records, schema.Record{
					RR:     *record.Name,
					Type:   *record.Type,
					Value:  *record.Value,
					Status: *record.Status,
				})
			}
			list = append(list, _domain)
		}
	}
	return list, nil
}
