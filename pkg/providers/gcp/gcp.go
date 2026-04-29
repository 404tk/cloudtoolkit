package gcp

import (
	"context"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/auth"
	_compute "github.com/404tk/cloudtoolkit/pkg/providers/gcp/compute"
	_dns "github.com/404tk/cloudtoolkit/pkg/providers/gcp/dns"
	_iam "github.com/404tk/cloudtoolkit/pkg/providers/gcp/iam"
	"github.com/404tk/cloudtoolkit/pkg/providers/internal/credverify"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
)

// Provider is a data provider for gcp API
type Provider struct {
	cred        auth.Credential
	tokenSource *auth.TokenSource
	apiClient   *api.Client
	projects    []string
}

// ClientConfig allows callers (e.g. demo replay) to inject a custom HTTP
// client used by both the OAuth2 token source and the API client, and skip
// credential cache writes for ephemeral credentials.
type ClientConfig struct {
	HTTPClient          *http.Client
	SkipCredentialCache bool
}

// New creates a new provider client for gcp API
func New(options schema.Options) (*Provider, error) {
	return NewWithConfig(options, ClientConfig{})
}

// NewWithConfig creates a new provider client for gcp API with an injected
// HTTP transport. Real callers use New; replay/test callers feed in a mock
// HTTP client through cfg.HTTPClient.
func NewWithConfig(options schema.Options, cfg ClientConfig) (*Provider, error) {
	cred, err := auth.FromOptions(options)
	if err != nil {
		return nil, err
	}
	if err := cred.Validate(); err != nil {
		return nil, err
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = api.NewHTTPClient()
	}
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

	if err := credverify.ForCloudlist(options, provider, cfg.SkipCredentialCache, func(context.Context) (credverify.Result, error) {
		return credverify.Result{
			Summary:     "Current project: " + cred.ProjectID,
			SessionUser: cred.ProjectID,
		}, nil
	}); err != nil {
		return nil, err
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
	collector := schema.NewResourceCollector(p.Name()).
		Register("host", func(ctx context.Context, list *schema.Resources) {
			instanceProvider := &_compute.Driver{Projects: p.projects, Client: p.apiClient}
			computes, err := instanceProvider.GetResource(ctx)
			schema.AppendAssets(list, computes)
			list.AddError("host", err)
		}).
		Register("domain", func(ctx context.Context, list *schema.Resources) {
			cloudDNSProvider := &_dns.Driver{Projects: p.projects, Client: p.apiClient}
			domains, err := cloudDNSProvider.GetDomains(ctx)
			schema.AppendAssets(list, domains)
			list.AddError("domain", err)
		}).
		Register("account", func(ctx context.Context, list *schema.Resources) {
			saProvider := &_iam.Driver{Projects: p.projects, Client: p.apiClient}
			users, err := saProvider.ListUsers(ctx)
			schema.AppendAssets(list, users)
			list.AddError("account", err)
		})

	return collector.Collect(ctx, env.From(ctx).Cloudlist)
}
