package gcp

import (
	"context"
	"encoding/base64"
	"errors"
	"log"

	_compute "github.com/404tk/cloudtoolkit/pkg/providers/gcp/compute"
	_dns "github.com/404tk/cloudtoolkit/pkg/providers/gcp/dns"
	_iam "github.com/404tk/cloudtoolkit/pkg/providers/gcp/iam"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/dns/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

// Provider is a data provider for gcp API
type Provider struct {
	vendor         string
	projects       []string
	dnsService     *dns.Service
	computeService *compute.Service
	iamService     *iam.Service
}

// New creates a new provider client for gcp API
func New(options schema.Options) (*Provider, error) {
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
	resp, err := manager.Projects.List().Do()
	if err != nil {
		return nil, err
	}
	for _, project := range resp.Projects {
		projects = append(projects, project.ProjectId)
	}
	if len(projects) > 0 {
		cache.Cfg.CredInsert(projects[0], options)
	} else {
		return nil, errors.New("[-] No project found.")
	}

	dnsService, err := dns.NewService(context.Background(), creds)
	if err != nil {
		return nil, err
	}
	computeService, _ := compute.NewService(context.Background(), creds)
	if err != nil {
		return nil, err
	}
	iamService, err := iam.NewService(context.Background(), creds)
	if err != nil {
		return nil, err
	}

	return &Provider{
		vendor:         "gcp",
		projects:       projects,
		dnsService:     dnsService,
		computeService: computeService,
		iamService:     iamService,
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
	var err error
	cloudDNSProvider := &_dns.CloudDNSProvider{Dns: p.dnsService, Projects: p.projects}
	list.Hosts, err = cloudDNSProvider.GetResource(ctx)

	InstanceProvider := &_compute.InstanceProvider{ComputeService: p.computeService, Projects: p.projects}
	computes, _ := InstanceProvider.GetResource(ctx)
	list.Hosts = append(list.Hosts, computes...)

	saProvider := &_iam.ServiceAccountProvider{IamService: p.iamService, Projects: p.projects}
	list.Users, err = saProvider.GetServiceAccounts(ctx)

	return list, err
}

func (p *Provider) UserManagement(action, uname, pwd string) {
	log.Println("[-] Not supported yet.")
}

func (p *Provider) BucketDump(action, bucketname string) {
	log.Println("[-] Not supported yet.")
}
