package azure

import (
	"context"
	"errors"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/azure/compute"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/storage"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

// Provider is a data provider for azure API
type Provider struct {
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

	var subscriptionIDs []string
	subscriptionID, ok := options.GetMetadata(utils.AzureSubscriptionId)
	if !ok {
		client := subscriptions.NewClient()
		client.Authorizer = authorizer
		resp, err := client.List(context.Background())
		if err != nil {
			return nil, err
		}
		for _, v := range resp.Values() {
			payload, _ := options.GetMetadata(utils.Payload)
			if payload == "cloudlist" {
				logger.Warning(fmt.Sprintf("Found Subscription: %s(%s)", *v.DisplayName, *v.SubscriptionID))
				cache.Cfg.CredInsert(*v.DisplayName, options)
			}
			subscriptionIDs = append(subscriptionIDs, *v.SubscriptionID)
		}
	} else {
		subscriptionIDs = append(subscriptionIDs, subscriptionID)
	}
	if len(subscriptionIDs) == 0 || subscriptionIDs[0] == "" {
		return nil, errors.New("No Subscription found.")
	}

	return &Provider{
		SubscriptionIDs:   subscriptionIDs,
		Authorizer:        authorizer,
		CredentialsConfig: config,
	}, nil
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return "azure"
}

// Resources returns the provider for an resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	list := schema.NewResources()
	list.Provider = p.Name()
	var err error
	for _, product := range utils.Cloudlist {
		switch product {
		case "host":
			vmProvider := &compute.Driver{SubscriptionIDs: p.SubscriptionIDs, Authorizer: p.Authorizer}
			list.Hosts, err = vmProvider.GetResource(ctx)
		case "bucket":
			storageProvider := &storage.Driver{
				SubscriptionIDs: p.SubscriptionIDs, Authorizer: p.Authorizer}
			list.Storages, err = storageProvider.GetStorages(ctx)
		default:
		}
	}

	// adProvider := activeDirectory.ADProvider{Config: p.CredentialsConfig}
	// list.Users, err = adProvider.GetActiveDirectory(ctx)

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
