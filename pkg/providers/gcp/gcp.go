package gcp

import (
	"context"
	"encoding/base64"

	_compute "github.com/404tk/cloudtoolkit/pkg/providers/gcp/compute"
	_dns "github.com/404tk/cloudtoolkit/pkg/providers/gcp/dns"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/dns/v1"
	"google.golang.org/api/option"
)

// Provider is a data provider for gcp API
type Provider struct {
	vendor         string
	projects       []string
	dnsService     *dns.Service
	computeService *compute.Service
}

// New creates a new provider client for gcp API
func New(options schema.OptionBlock) (*Provider, error) {
	gcpKey, ok := options.GetMetadata(utils.GCPserviceAccountJSON)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: utils.GCPserviceAccountJSON}
	}
	tojson, err := base64.StdEncoding.DecodeString(gcpKey)
	if err != nil {
		return nil, err
	}

	creds := option.WithCredentialsJSON(tojson)

	projects := []string{}
	manager, err := cloudresourcemanager.NewService(context.Background(), creds)
	if err != nil {
		return nil, err
	}
	list := manager.Projects.List()
	err = list.Pages(context.Background(), func(resp *cloudresourcemanager.ListProjectsResponse) error {
		for _, project := range resp.Projects {
			projects = append(projects, project.ProjectId)
		}
		return nil
	})

	dnsService, err := dns.NewService(context.Background(), creds)
	if err != nil {
		return nil, err
	}
	computeService, err := compute.NewService(context.Background(), creds)
	if err != nil {
		return nil, err
	}

	return &Provider{
		vendor:         "gcp",
		projects:       projects,
		dnsService:     dnsService,
		computeService: computeService,
	}, err
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return p.vendor
}

// Resources returns the provider for an resource deployment source.
func (p *Provider) Resources(ctx context.Context) (*schema.Resources, error) {
	list := schema.NewResources()
	list.Provider = p.vendor

	cloudDNSProvider := &_dns.CloudDNSProvider{Dns: p.dnsService, Projects: p.projects}
	dnshosts, _ := cloudDNSProvider.GetResource(ctx)
	list.Hosts = append(list.Hosts, dnshosts...)

	InstanceProvider := &_compute.InstanceProvider{ComputeService: p.computeService, Projects: p.projects}
	computes, _ := InstanceProvider.GetResource(ctx)
	list.Hosts = append(list.Hosts, computes...)

	return list, nil
}

func (p *Provider) UserManagement(action, uname, pwd string) {}
