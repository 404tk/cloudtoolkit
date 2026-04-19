package volcengine

import (
	"context"

	_api "github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	_auth "github.com/404tk/cloudtoolkit/pkg/providers/volcengine/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/billing"
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/ecs"
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/iam"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Provider struct {
	region    string
	apiClient *_api.Client
}

// New creates a new provider client for volcengine API.
func New(options schema.Options) (*Provider, error) {
	credential, err := _auth.FromOptions(options)
	if err != nil {
		return nil, err
	}
	region, _ := options.GetMetadata(utils.Region)
	apiClient := _api.NewClient(credential)

	payload, _ := options.GetMetadata(utils.Payload)
	if payload == "cloudlist" {
		name, err := (&iam.Driver{Client: apiClient, Region: region}).GetProject(context.Background())
		if err != nil {
			return nil, err
		}
		logger.Warning("Current project:", name)
		cache.Cfg.CredInsert(name, options)
	}

	return &Provider{
		region:    region,
		apiClient: apiClient,
	}, nil
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return "volcengine"
}

// Resources returns the provider for a resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	list := schema.NewResources()
	list.Provider = p.Name()
	for _, product := range utils.Cloudlist {
		switch product {
		case "balance":
			(&billing.Driver{Client: p.apiClient, Region: p.region}).QueryAccountBalance(ctx)
		case "host":
			d := &ecs.Driver{Client: p.apiClient, Region: p.region}
			hosts, err := d.GetResource(ctx)
			schema.AppendAssets(&list, hosts)
			list.AddError("host", err)
		case "domain":
		case "account":
			d := &iam.Driver{Client: p.apiClient, Region: p.region}
			users, err := d.ListUsers(ctx)
			schema.AppendAssets(&list, users)
			list.AddError("account", err)
		case "database":
		case "bucket":
		case "sms":
		case "log":
		default:
		}
	}

	return list, list.Err()
}
