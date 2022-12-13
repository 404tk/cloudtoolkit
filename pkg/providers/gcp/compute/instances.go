package compute

import (
	"context"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"google.golang.org/api/compute/v1"
)

type InstanceProvider struct {
	ComputeService *compute.Service
	Projects       []string
}

func (d *InstanceProvider) GetResource(ctx context.Context) ([]*schema.Host, error) {
	list := schema.NewResources().Hosts
	log.Println("[*] Start enumerating Compute ...")

	for _, project := range d.Projects {
		resp, err := d.ComputeService.Zones.List(project).Do()
		if err != nil {
			log.Printf("[-] Could not get all zones for project %s.\n", project)
			return list, err
		}
		for _, z := range resp.Items {
			res, err := d.ComputeService.Instances.List(project, z.Name).Context(ctx).Do()
			if err != nil {
				log.Printf("[-] Could not list instances for zone %s in project %s.\n", z.Name, project)
				return list, err
			}
			for _, instance := range res.Items {
				_host := &schema.Host{Region: instance.Zone}
				for _, networkInterface := range instance.NetworkInterfaces {
					_host.PrivateIpv4 = networkInterface.NetworkIP
					for _, accessConfig := range networkInterface.AccessConfigs {
						if accessConfig.NatIP != "" {
							_host.Public = true
							_host.PublicIPv4 = accessConfig.NatIP
							goto save
						}
					}
				}
			save:
				list = append(list, _host)
			}
		}
	}
	return list, nil
}
