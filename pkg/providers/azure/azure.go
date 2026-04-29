package azure

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	azapi "github.com/404tk/cloudtoolkit/pkg/providers/azure/api"
	azauth "github.com/404tk/cloudtoolkit/pkg/providers/azure/auth"
	azcloud "github.com/404tk/cloudtoolkit/pkg/providers/azure/cloud"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/compute"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/storage"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

// Provider is a data provider for Azure ARM APIs.
type Provider struct {
	cred            azauth.Credential
	endpoints       azcloud.Endpoints
	tokenSource     *azauth.TokenSource
	apiClient       *azapi.Client
	subscriptionIDs []string
}

// ClientConfig allows callers (e.g. demo replay) to inject a custom HTTP
// client used by both the OAuth2 token source and the ARM API client, and
// skip credential cache writes for ephemeral credentials.
type ClientConfig struct {
	HTTPClient          *http.Client
	SkipCredentialCache bool
}

// New creates a new provider client for Azure API.
func New(options schema.Options) (*Provider, error) {
	return NewWithConfig(options, ClientConfig{})
}

// NewWithConfig creates a new provider client for Azure API with an injected
// HTTP transport. Real callers use New; replay/test callers feed in a mock
// HTTP client through cfg.HTTPClient.
func NewWithConfig(options schema.Options, cfg ClientConfig) (*Provider, error) {
	cred, err := azauth.FromOptions(options)
	if err != nil {
		return nil, err
	}
	if err := cred.Validate(); err != nil {
		return nil, err
	}

	endpoints := azcloud.For(cred.Cloud)
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = azapi.NewHTTPClient()
	}
	tokenSource := azauth.NewTokenSource(cred, httpClient)
	client := azapi.NewClient(tokenSource, endpoints, azapi.WithHTTPClient(httpClient))
	provider := &Provider{
		cred:        cred,
		endpoints:   endpoints,
		tokenSource: tokenSource,
		apiClient:   client,
	}

	subscriptionIDs := make([]string, 0, 1)
	if cred.SubscriptionID != "" {
		subscriptionIDs = append(subscriptionIDs, cred.SubscriptionID)
	} else {
		pager := azapi.NewPager[azapi.Subscription](client, azapi.Request{
			Method:     http.MethodGet,
			Path:       "/subscriptions",
			Query:      url.Values{"api-version": {azapi.SubscriptionsAPIVersion}},
			Idempotent: true,
		})
		allSubscriptions, err := pager.All(context.Background())
		if err != nil {
			return nil, err
		}
		payload, _ := options.GetMetadata(utils.Payload)
		for _, sub := range allSubscriptions {
			if payload == "cloudlist" {
				logger.Warning(fmt.Sprintf("Found Subscription: %s(%s)", sub.DisplayName, sub.SubscriptionID))
				if !cfg.SkipCredentialCache {
					cache.Cfg.CredInsert(sub.DisplayName, provider, options)
				}
			}
			if sub.SubscriptionID != "" {
				subscriptionIDs = append(subscriptionIDs, sub.SubscriptionID)
			}
		}
	}

	if len(subscriptionIDs) == 0 || subscriptionIDs[0] == "" {
		return nil, errors.New("no subscription found")
	}

	provider.subscriptionIDs = subscriptionIDs
	return provider, nil
}

// Name returns the name of the provider.
func (p *Provider) Name() string {
	return "azure"
}

func (p *Provider) CredentialKey(opts map[string]string) string {
	return opts[utils.AzureClientId]
}

// Resources returns the provider for a resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	collector := schema.NewResourceCollector(p.Name()).
		Register("host", func(ctx context.Context, list *schema.Resources) {
			vmProvider := &compute.Driver{
				Client:          p.apiClient,
				SubscriptionIDs: p.subscriptionIDs,
			}
			hosts, err := vmProvider.GetResource(ctx)
			schema.AppendAssets(list, hosts)
			list.AddError("host", err)
		}).
		Register("bucket", func(ctx context.Context, list *schema.Resources) {
			storageProvider := &storage.Driver{
				Client:          p.apiClient,
				SubscriptionIDs: p.subscriptionIDs,
			}
			storages, err := storageProvider.GetStorages(ctx)
			schema.AppendAssets(list, storages)
			list.AddError("bucket", err)
		})

	return collector.Collect(ctx, env.From(ctx).Cloudlist)
}
