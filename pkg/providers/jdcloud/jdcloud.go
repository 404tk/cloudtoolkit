package jdcloud

import (
	"context"
	"fmt"

	_api "github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	_auth "github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/iam"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/oss"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/vm"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
)

type Provider struct {
	region    string
	apiClient *_api.Client
}

// New creates a new provider client for JDCloud API.
func New(options schema.Options) (*Provider, error) {
	credential, err := _auth.FromOptions(options)
	if err != nil {
		return nil, err
	}
	region, _ := options.GetMetadata(utils.Region)
	apiClient := _api.NewClient(credential)
	payload, _ := options.GetMetadata(utils.Payload)

	if payload == "cloudlist" {
		d := &iam.Driver{Client: apiClient}
		if !d.Validator(credential.AccessKey) {
			return nil, fmt.Errorf("invalid accesskey")
		}
		cache.Cfg.CredInsert("default", options)
	}

	return &Provider{
		region:    region,
		apiClient: apiClient,
	}, nil
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return "jdcloud"
}

// Resources returns the provider for a resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	list := schema.NewResources()
	list.Provider = p.Name()
	for _, product := range utils.Cloudlist {
		switch product {
		case "balance":
		case "host":
			d := &vm.Driver{Client: p.apiClient, Region: p.region}
			hosts, err := d.GetResource(ctx)
			schema.AppendAssets(&list, hosts)
			list.AddError("host", err)
		case "domain":
		case "account":
			d := &iam.Driver{Client: p.apiClient}
			users, err := d.ListUsers(ctx)
			schema.AppendAssets(&list, users)
			list.AddError("account", err)
		case "database":
		case "bucket":
			d := &oss.Driver{Client: p.apiClient}
			storages, err := d.ListBuckets(ctx)
			schema.AppendAssets(&list, storages)
			list.AddError("bucket", err)
		case "sms":
		case "log":
		default:
		}
	}

	return list, list.Err()
}
