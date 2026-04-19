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

// New creates a new provider client for Azure API.
func New(options schema.Options) (*Provider, error) {
	cred, err := azauth.FromOptions(options)
	if err != nil {
		return nil, err
	}
	if err := cred.Validate(); err != nil {
		return nil, err
	}

	endpoints := azcloud.For(cred.Cloud)
	httpClient := azapi.NewHTTPClient()
	tokenSource := azauth.NewTokenSource(cred, httpClient)
	client := azapi.NewClient(tokenSource, endpoints, azapi.WithHTTPClient(httpClient))

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
				cache.Cfg.CredInsert(sub.DisplayName, options)
			}
			if sub.SubscriptionID != "" {
				subscriptionIDs = append(subscriptionIDs, sub.SubscriptionID)
			}
		}
	}

	if len(subscriptionIDs) == 0 || subscriptionIDs[0] == "" {
		return nil, errors.New("No Subscription found.")
	}

	return &Provider{
		cred:            cred,
		endpoints:       endpoints,
		tokenSource:     tokenSource,
		apiClient:       client,
		subscriptionIDs: subscriptionIDs,
	}, nil
}

// Name returns the name of the provider.
func (p *Provider) Name() string {
	return "azure"
}

// Resources returns the provider for a resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	list := schema.NewResources()
	list.Provider = p.Name()

	for _, product := range utils.Cloudlist {
		switch product {
		case "host":
			vmProvider := &compute.Driver{
				Client:          p.apiClient,
				SubscriptionIDs: p.subscriptionIDs,
			}
			hosts, err := vmProvider.GetResource(ctx)
			schema.AppendAssets(&list, hosts)
			list.AddError("host", err)
		case "bucket":
			storageProvider := &storage.Driver{
				Client:          p.apiClient,
				SubscriptionIDs: p.subscriptionIDs,
			}
			storages, err := storageProvider.GetStorages(ctx)
			schema.AppendAssets(&list, storages)
			list.AddError("bucket", err)
		default:
		}
	}

	return list, list.Err()
}
