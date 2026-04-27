package gcp

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/auth"
	_compute "github.com/404tk/cloudtoolkit/pkg/providers/gcp/compute"
	_dns "github.com/404tk/cloudtoolkit/pkg/providers/gcp/dns"
	_iam "github.com/404tk/cloudtoolkit/pkg/providers/gcp/iam"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

// Provider is a data provider for gcp API
type Provider struct {
	cred        auth.Credential
	tokenSource *auth.TokenSource
	apiClient   *api.Client
	projects    []string
}

// New creates a new provider client for gcp API
func New(options schema.Options) (*Provider, error) {
	cred, err := auth.FromOptions(options)
	if err != nil {
		return nil, err
	}
	if err := cred.Validate(); err != nil {
		return nil, err
	}

	httpClient := api.NewHTTPClient()
	ts := auth.NewTokenSource(cred, httpClient)
	client := api.NewClient(ts, api.WithHTTPClient(httpClient))
	provider := &Provider{
		cred:        cred,
		tokenSource: ts,
		apiClient:   client,
		projects:    []string{cred.ProjectID},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := ts.Token(ctx); err != nil {
		return nil, err
	}

	payload, _ := options.GetMetadata(utils.Payload)
	if payload == "cloudlist" {
		logger.Warning("Current project:", cred.ProjectID)
		cache.Cfg.CredInsert(cred.ProjectID, provider, options)
	}

	return provider, nil
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return "gcp"
}

func (p *Provider) CredentialKey(opts map[string]string) string {
	tojson, _ := base64.StdEncoding.DecodeString(opts[utils.GCPserviceAccountJSON])
	return utils.Md5Encode(string(tojson))
}

// Resources returns the provider for an resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	list := schema.NewResources()
	list.Provider = p.Name()
	for _, product := range env.From(ctx).Cloudlist {
		switch product {
		case "host":
			InstanceProvider := &_compute.Driver{Projects: p.projects, Client: p.apiClient}
			computes, err := InstanceProvider.GetResource(ctx)
			schema.AppendAssets(&list, computes)
			list.AddError("host", err)
		case "domain":
			cloudDNSProvider := &_dns.Driver{Projects: p.projects, Client: p.apiClient}
			domains, err := cloudDNSProvider.GetDomains(ctx)
			schema.AppendAssets(&list, domains)
			list.AddError("domain", err)
		case "account":
			saProvider := &_iam.Driver{Projects: p.projects, Client: p.apiClient}
			users, err := saProvider.ListUsers(ctx)
			schema.AppendAssets(&list, users)
			list.AddError("account", err)
		default:
		}
	}

	return list, list.Err()
}
