package dns

import (
	"context"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"google.golang.org/api/dns/v1"
)

type CloudDNSProvider struct {
	Dns      *dns.Service
	Projects []string
}

func (d *CloudDNSProvider) GetResource(ctx context.Context) ([]*schema.Host, error) {
	list := schema.NewResources().Hosts

	for _, project := range d.Projects {
		zone := d.Dns.ManagedZones.List(project)
		err := zone.Pages(context.Background(), func(resp *dns.ManagedZonesListResponse) error {
			for _, z := range resp.ManagedZones {
				resources := d.Dns.ResourceRecordSets.List(project, z.Name)
				err := resources.Pages(context.Background(), func(r *dns.ResourceRecordSetsListResponse) error {
					items := d.parseRecordsForResourceSet(r)
					list = append(list, items...)
					return nil
				})
				if err != nil {
					log.Printf("Could not get resource_records for zone %s in project %s: %s\n", z.Name, project, err.Error())
					continue
				}
			}
			return nil
		})
		if err != nil {
			log.Printf("Could not get all zones for project %s: %s\n", project, err.Error())
			continue
		}
	}
	return list, nil
}

// parseRecordsForResourceSet parses and returns the records for a resource set
func (d *CloudDNSProvider) parseRecordsForResourceSet(r *dns.ResourceRecordSetsListResponse) []*schema.Host {
	list := schema.NewResources().Hosts

	for _, resource := range r.Rrsets {
		if resource.Type != "A" && resource.Type != "CNAME" && resource.Type != "AAAA" {
			continue
		}

		for _, data := range resource.Rrdatas {
			list = append(list, &schema.Host{
				DNSName:    resource.Name,
				Public:     true,
				PublicIPv4: data,
			})
		}
	}
	return list
}
