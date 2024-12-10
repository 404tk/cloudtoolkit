package jdcloud

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/iam"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/oss"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/vm"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/jdcloud-api/jdcloud-sdk-go/core"
)

type Provider struct {
	cred   *core.Credential
	token  string
	region string
}

// New creates a new provider client for alibaba API
func New(options schema.Options) (*Provider, error) {
	accessKey, ok := options.GetMetadata(utils.AccessKey)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: utils.AccessKey}
	}
	secretKey, ok := options.GetMetadata(utils.SecretKey)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: utils.SecretKey}
	}
	region, _ := options.GetMetadata(utils.Region)
	token, _ := options.GetMetadata(utils.SecurityToken)
	cred := core.NewCredentials(accessKey, secretKey)
	payload, _ := options.GetMetadata(utils.Payload)

	if payload == "cloudlist" {
		d := &iam.Driver{Cred: cred, Token: token}
		if !d.Validator(accessKey) {
			return nil, fmt.Errorf("Invalid Accesskey")
		}
		cache.Cfg.CredInsert("default", options)
	}

	return &Provider{
		cred:   cred,
		token:  token,
		region: region,
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
	var err error
	for _, product := range utils.Cloudlist {
		switch product {
		case "balance":
		case "host":
			d := &vm.Driver{Cred: p.cred, Token: p.token, Region: p.region}
			list.Hosts, err = d.GetResource(ctx)
		case "domain":
		case "account":
			d := &iam.Driver{Cred: p.cred, Token: p.token}
			list.Users, err = d.ListUsers(ctx)
		case "database":
		case "bucket":
			d := &oss.Driver{Cred: p.cred, Token: p.token}
			list.Storages, err = d.ListBuckets(ctx)
		case "sms":
		case "log":
		default:
		}
	}

	return list, err
}

func (p *Provider) UserManagement(action, args_1, args_2 string) {}

func (p *Provider) BucketDump(ctx context.Context, action, bucketname string) {}

func (p *Provider) EventDump(action, args string) {}

func (p *Provider) ExecuteCloudVMCommand(instanceId, cmd string) {}

func (p *Provider) DBManagement(action, args string) {}
