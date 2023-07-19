package azure

import (
	"context"
	"errors"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/providers/azure/compute"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/storage"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

// Provider is a data provider for azure API
type Provider struct {
	vendor            string
	SubscriptionIDs   []string
	Authorizer        autorest.Authorizer
	CredentialsConfig auth.ClientCredentialsConfig
}

// New creates a new provider client for azure API
func New(options schema.Options) (*Provider, error) {
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

	var subscription_ids []string
	subscriptionID, ok := options.GetMetadata(utils.AzureSubscriptionId)
	if !ok {
		client := subscriptions.NewClient()
		client.Authorizer = authorizer
		resp, err := client.List(context.Background())
		if err != nil {
			return nil, err
		}
		for _, v := range resp.Values() {
			log.Printf("[+] Found Subscription: %s(%s)\n", *v.DisplayName, *v.SubscriptionID)
			cache.Cfg.CredInsert(*v.DisplayName, options)
			subscription_ids = append(subscription_ids, *v.SubscriptionID)
		}
	} else {
		subscription_ids = append(subscription_ids, subscriptionID)
	}
	if len(subscription_ids) == 0 || subscription_ids[0] == "" {
		return nil, errors.New("[-] No Subscription found.")
	}

	return &Provider{
		vendor:            "azure",
		SubscriptionIDs:   subscription_ids,
		Authorizer:        authorizer,
		CredentialsConfig: config,
	}, nil
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return p.vendor
}

// Resources returns the provider for an resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	list := schema.NewResources()
	list.Provider = p.vendor
	var err error
	vmProvider := &compute.Driver{SubscriptionIDs: p.SubscriptionIDs, Authorizer: p.Authorizer}
	list.Hosts, err = vmProvider.GetResource(ctx)

	storageProvider := &storage.Driver{
		SubscriptionIDs: p.SubscriptionIDs, Authorizer: p.Authorizer}
	list.Storages, err = storageProvider.GetStorages(ctx)

	// adProvider := activeDirectory.ADProvider{Config: p.CredentialsConfig}
	// list.Users, err = adProvider.GetActiveDirectory(ctx)

	return list, err
}

func (p *Provider) UserManagement(action, uname, pwd string) {
	log.Println("[-] Not supported yet.")
}

func (p *Provider) BucketDump(ctx context.Context, action, bucketname string) {
	log.Println("[-] Not supported yet.")
}

func (p *Provider) EventDump(action, sourceIp string) {}
