package azure

import (
	"context"

	"github.com/404tk/cloudtoolkit/pkg/providers/azure/compute"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

// Provider is a data provider for azure API
type Provider struct {
	vendor            string
	SubscriptionID    string
	Authorizer        autorest.Authorizer
	CredentialsConfig auth.ClientCredentialsConfig
}

// New creates a new provider client for azure API
func New(options schema.OptionBlock) (*Provider, error) {
	subscriptionID, ok := options.GetMetadata(utils.AzureSubscriptionId)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: utils.AzureSubscriptionId}
	}
	clientID, ok := options.GetMetadata(utils.AzureClientId)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: utils.AzureClientId}
	}
	clientSecret, ok := options.GetMetadata(utils.AzureClientSecret)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: utils.AzureClientSecret}
	}
	tenantID, ok := options.GetMetadata(utils.AzureTenantId)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: utils.AzureTenantId}
	}

	config := auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, err
	}

	return &Provider{
		vendor:            "azure",
		SubscriptionID:    subscriptionID,
		Authorizer:        authorizer,
		CredentialsConfig: config,
	}, nil
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return p.vendor
}

// Resources returns the provider for an resource deployment source.
func (p *Provider) Resources(ctx context.Context) (*schema.Resources, error) {
	list := schema.NewResources()
	list.Provider = p.vendor
	vmProvider := &compute.VmProvider{SubscriptionID: p.SubscriptionID, Authorizer: p.Authorizer}
	list.Hosts, _ = vmProvider.GetResource(ctx)

	return list, nil
}
