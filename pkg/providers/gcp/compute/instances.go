package compute

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Projects []string
	Client   *api.Client
}

func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := []schema.Host{}
	logger.Info("List Compute ...")
	for _, project := range d.Projects {
		zones, err := d.listZones(ctx, project)
		if err != nil {
			logger.Error(fmt.Sprintf("List %s zones failed: %s.", project, err.Error()))
			return list, err
		}
		for _, z := range zones {
			instances, err := d.listInstances(ctx, project, z.Name)
			if err != nil {
				logger.Error(fmt.Sprintf("List projects/%s/zones/%s/instances failed: %s", project, z.Name, err.Error()))
				return list, err
			}
			for _, i := range instances {
				zoneShort := shortResourceName(i.Zone)
				instanceName := strings.TrimSpace(i.Name)
				hostName := strings.TrimSpace(i.Hostname)
				if hostName == "" {
					hostName = instanceName
				}
				_host := schema.Host{
					HostName: hostName,
					ID:       composeInstanceID(zoneShort, instanceName),
					Region:   zoneShort,
				}
				foundPublic := false
				for _, n := range i.NetworkInterfaces {
					_host.PrivateIpv4 = n.NetworkIP
					for _, acc := range n.AccessConfigs {
						natIP := acc.NatIP
						if natIP != "" {
							_host.Public = true
							_host.PublicIPv4 = natIP
							foundPublic = true
							break
						}
					}
					if foundPublic {
						break
					}
				}
				list = append(list, _host)
			}
		}
	}
	return list, nil
}

// shortResourceName trims a Compute Engine self-link down to the trailing
// resource name. Zone fields can come back either short ("us-central1-a") or
// fully-qualified ("https://www.googleapis.com/compute/v1/projects/p/zones/us-central1-a").
func shortResourceName(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.LastIndex(s, "/"); idx >= 0 {
		return s[idx+1:]
	}
	return s
}

// composeInstanceID returns the cloudlist Host.ID form expected by vmexec:
// `<zone>/<instance>` so REPL `shell <id>` and headless `--target` round-trip
// through `vmexec.resolveTarget` deterministically.
func composeInstanceID(zone, instance string) string {
	if instance == "" {
		return ""
	}
	if zone == "" {
		return instance
	}
	return zone + "/" + instance
}

func (d *Driver) listZones(ctx context.Context, project string) ([]api.Zone, error) {
	pager := api.NewPager[api.Zone](d.Client, api.Request{
		Method:     http.MethodGet,
		BaseURL:    api.ComputeBaseURL,
		Path:       "/compute/v1/projects/" + url.PathEscape(project) + "/zones",
		Idempotent: true,
	}, "items")
	return pager.All(ctx)
}

func (d *Driver) listInstances(ctx context.Context, project, zone string) ([]api.Instance, error) {
	pager := api.NewPager[api.Instance](d.Client, api.Request{
		Method:     http.MethodGet,
		BaseURL:    api.ComputeBaseURL,
		Path:       "/compute/v1/projects/" + url.PathEscape(project) + "/zones/" + url.PathEscape(zone) + "/instances",
		Idempotent: true,
	}, "items")
	return pager.All(ctx)
}
