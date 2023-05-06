package gcp

import (
	"context"
	"encoding/base64"
	"log"
	"time"

	_compute "github.com/404tk/cloudtoolkit/pkg/providers/gcp/compute"
	_dns "github.com/404tk/cloudtoolkit/pkg/providers/gcp/dns"
	_iam "github.com/404tk/cloudtoolkit/pkg/providers/gcp/iam"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"golang.org/x/oauth2/google"
)

// Provider is a data provider for gcp API
type Provider struct {
	vendor   string
	projects []string
	token    string
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

	var projects []string
	var token string
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	credentials, err := google.CredentialsFromJSON(ctx, tojson, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, err
	}
	if credentials != nil {
		projects = append(projects, credentials.ProjectID)
		access, err := credentials.TokenSource.Token()
		if err != nil {
			return nil, err
		}
		token = access.AccessToken
	}

	/*
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
	*/

	return &Provider{
		vendor:   "gcp",
		projects: projects,
		token:    token,
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
	cloudDNSProvider := &_dns.CloudDNSProvider{Projects: p.projects, Token: p.token}
	list.Hosts, err = cloudDNSProvider.GetResource(ctx)

	InstanceProvider := &_compute.InstanceProvider{Projects: p.projects, Token: p.token}
	computes, _ := InstanceProvider.GetResource(ctx)
	list.Hosts = append(list.Hosts, computes...)

	saProvider := &_iam.ServiceAccountProvider{Projects: p.projects, Token: p.token}
	list.Users, err = saProvider.GetServiceAccounts(ctx)

	return list, err
}

func (p *Provider) UserManagement(action, uname, pwd string) {
	log.Println("[-] Not supported yet.")
}

func (p *Provider) BucketDump(action, bucketname string) {
	log.Println("[-] Not supported yet.")
}
