package dns

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/request"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/tidwall/gjson"
)

type Driver struct {
	Projects []string
	Token    string
}

func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := schema.NewResources().Hosts
	logger.Info("Start enumerating DNS ...")
	r := &request.DefaultHttpRequest{
		Endpoint: "dns.googleapis.com",
		Method:   "GET",
		Token:    d.Token,
	}

	for _, project := range d.Projects {
		zones, err := r.ListManagedZones(project)
		if err != nil {
			logger.Error(fmt.Sprintf("List %s zones failed: %s.\n", project, err.Error()))
			return list, err
		}
		for _, z := range zones {
			resources, err := r.ListRRSets(project, z)
			if err != nil {
				logger.Error(fmt.Sprintf("List projects/%s/managedZones/%s/rrsets failed: %s\n", project, z, err.Error()))
				return list, err
			}
			items := d.parseRecordsForResourceSet(resources, z)
			list = append(list, items...)
		}
	}

	return list, nil
}

// parseRecordsForResourceSet parses and returns the records for a resource set
func (d *Driver) parseRecordsForResourceSet(r []gjson.Result, zone string) []schema.Host {
	list := schema.NewResources().Hosts

	for _, resource := range r {
		_type := resource.Get("type").String()
		if _type != "A" && _type != "CNAME" && _type != "AAAA" {
			continue
		}

		name := resource.Get("name").String()
		datas := resource.Get("rrdatas").Array()
		for _, data := range datas {
			list = append(list, schema.Host{
				DNSName:    name,
				Public:     true,
				PublicIPv4: data.String(),
				Region:     zone,
			})
		}
	}
	return list
}
