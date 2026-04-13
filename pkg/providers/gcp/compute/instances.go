package compute

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

func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := []schema.Host{}
	logger.Info("List Compute ...")
	r := &request.DefaultHttpRequest{
		Endpoint: "compute.googleapis.com",
		Method:   "GET",
		Token:    d.Token,
	}
	for _, project := range d.Projects {
		zones, err := r.ListZones(project)
		if err != nil {
			logger.Error(fmt.Sprintf("List %s zones failed: %s.", project, err.Error()))
			return list, err
		}
		for _, z := range zones {
			instances, err := r.ListInstances(project, z)
			if err != nil {
				logger.Error(fmt.Sprintf("List projects/%s/zones/%s/instances failed: %s", project, z, err.Error()))
				return list, err
			}
			for _, i := range instances {
				_host := schema.Host{
					HostName: i.Hostname,
					Region:   i.Zone,
				}
				for _, n := range i.NetworkInterfaces {
					_host.PrivateIpv4 = n.NetworkIP
					for _, acc := range n.AccessConfigs {
						natIP := acc.NatIP
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
