package gcp

import (
	"context"
	"encoding/base64"
	"time"

	_compute "github.com/404tk/cloudtoolkit/pkg/providers/gcp/compute"
	_dns "github.com/404tk/cloudtoolkit/pkg/providers/gcp/dns"
	_iam "github.com/404tk/cloudtoolkit/pkg/providers/gcp/iam"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"golang.org/x/oauth2/google"
)

// Provider is a data provider for gcp API
type Provider struct {
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
		payload, _ := options.GetMetadata(utils.Payload)
		if payload == "cloudlist" {
			logger.Warning("Current project:", projects[0])
			cache.Cfg.CredInsert(projects[0], options)
		}
	}

	return &Provider{
		projects: projects,
		token:    token,
	}, err
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return "gcp"
}

// Resources returns the provider for an resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	list := schema.NewResources()
	list.Provider = p.Name()
	var err error
	for _, product := range utils.Cloudlist {
		switch product {
		case "host":
			cloudDNSProvider := &_dns.Driver{Projects: p.projects, Token: p.token}
			list.Hosts, err = cloudDNSProvider.GetResource(ctx)
			InstanceProvider := &_compute.Driver{Projects: p.projects, Token: p.token}
			computes, _ := InstanceProvider.GetResource(ctx)
			list.Hosts = append(list.Hosts, computes...)
		case "account":
			saProvider := &_iam.Driver{Projects: p.projects, Token: p.token}
			list.Users, err = saProvider.ListUsers(ctx)
		default:
		}
	}

	return list, err
}

func (p *Provider) UserManagement(action, username, password string) {
	logger.Error("Not supported yet.")
}

func (p *Provider) BucketDump(ctx context.Context, action, bucketName string) {
	logger.Error("Not supported yet.")
}

func (p *Provider) EventDump(action, sourceIP string) {}

func (p *Provider) ExecuteCloudVMCommand(instanceID, cmd string) {}

func (p *Provider) DBManagement(action, instanceID string) {}
