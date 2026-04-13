package dns

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/request"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Projects []string
	Token    string
}

func (d *Driver) GetDomains(ctx context.Context) ([]schema.Domain, error) {
	list := []schema.Domain{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List DNS ...")
	}
	r := &request.DefaultHttpRequest{
		Endpoint: "dns.googleapis.com",
		Method:   "GET",
		Token:    d.Token,
	}

	for _, project := range d.Projects {
		zones, err := r.ListManagedZones(project)
		if err != nil {
			logger.Error(fmt.Sprintf("List %s zones failed: %s.", project, err.Error()))
			return list, err
		}
		for _, z := range zones {
			zoneName := z.Name
			resources, err := r.ListRRSets(project, zoneName)
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
func (d *Driver) parseRecordsForResourceSet(r []request.RRSet) []schema.Record {
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
