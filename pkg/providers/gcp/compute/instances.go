package compute

import (
	"context"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/request"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

type Driver struct {
	Projects []string
	Token    string
}

func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := schema.NewResources().Hosts
	log.Println("[*] Start enumerating Compute ...")
	r := &request.DefaultHttpRequest{
		Endpoint: "compute.googleapis.com",
		Method:   "GET",
		Token:    d.Token,
	}
	for _, project := range d.Projects {
		zones, err := r.ListZones(project)
		if err != nil {
			log.Printf("[-] List %s zones failed: %s.\n", project, err.Error())
			return list, err
		}
		for _, z := range zones {
			instances, err := r.ListInstances(project, z)
			if err != nil {
				log.Printf("[-] List projects/%s/zones/%s/instances failed: %s\n", project, z, err.Error())
				return list, err
			}
			for _, i := range instances {
				_host := schema.Host{Region: i.Get("zone").String()}
				network := i.Get("networkInterfaces").Array()
				for _, n := range network {
					_host.PrivateIpv4 = n.Get("networkIP").String()
					conf := n.Get("networkInterfaces").Array()
					for _, acc := range conf {
						natIP := acc.Get("natIP").String()
						if natIP != "" {
							_host.Public = true
							_host.PublicIPv4 = natIP
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
