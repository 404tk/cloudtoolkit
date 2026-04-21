package dns

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Projects []string
	Client   *api.Client
}

func (d *Driver) GetDomains(ctx context.Context) ([]schema.Domain, error) {
	list := []schema.Domain{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List DNS ...")
	}
	for _, project := range d.Projects {
		zones, err := d.listManagedZones(ctx, project)
		if err != nil {
			logger.Error(fmt.Sprintf("List %s zones failed: %s.", project, err.Error()))
			return list, err
		}
		for _, z := range zones {
			zoneName := z.Name
			resources, err := d.listRRSets(ctx, project, zoneName)
			if err != nil {
				logger.Error(fmt.Sprintf("List projects/%s/managedZones/%s/rrsets failed: %s", project, zoneName, err.Error()))
				return list, err
			}
			domainName := z.DNSName
			if domainName == "" {
				domainName = zoneName
			}
			records := d.parseRecordsForResourceSet(resources)
			list = append(list, schema.Domain{
				DomainName: domainName,
				Records:    records,
			})
		}
	}

	return list, nil
}

// parseRecordsForResourceSet parses and returns the records for a resource set
func (d *Driver) parseRecordsForResourceSet(r []api.RRSet) []schema.Record {
	list := []schema.Record{}

	for _, resource := range r {
		_type := resource.Type
		if _type != "A" && _type != "CNAME" && _type != "AAAA" {
			continue
		}

		name := resource.Name
		for _, data := range resource.RRDatas {
			list = append(list, schema.Record{
				RR:    name,
				Type:  _type,
				Value: data,
			})
		}
	}
	return list
}

func (d *Driver) listManagedZones(ctx context.Context, project string) ([]api.ManagedZone, error) {
	pager := api.NewPager[api.ManagedZone](d.Client, api.Request{
		Method:     http.MethodGet,
		BaseURL:    api.DNSBaseURL,
		Path:       "/dns/v1/projects/" + url.PathEscape(project) + "/managedZones",
		Idempotent: true,
	}, "managedZones")
	return pager.All(ctx)
}

func (d *Driver) listRRSets(ctx context.Context, project, zone string) ([]api.RRSet, error) {
	pager := api.NewPager[api.RRSet](d.Client, api.Request{
		Method:     http.MethodGet,
		BaseURL:    api.DNSBaseURL,
		Path:       "/dns/v1/projects/" + url.PathEscape(project) + "/managedZones/" + url.PathEscape(zone) + "/rrsets",
		Idempotent: true,
	}, "rrsets")
	return pager.All(ctx)
}
