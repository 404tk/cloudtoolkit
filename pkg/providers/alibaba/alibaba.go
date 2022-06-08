package alibaba

import (
	"context"

	_ecs "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/ecs"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
)

// Provider is a data provider for alibaba API
type Provider struct {
	vendor         string
	EcsClient      *ecs.Client
	resourceGroups []string
}

// New creates a new provider client for alibaba API
func New(options schema.OptionBlock) (*Provider, error) {
	accessKey, ok := options.GetMetadata(utils.AccessKey)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: utils.AccessKey}
	}
	secretKey, ok := options.GetMetadata(utils.SecretKey)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: utils.SecretKey}
	}
	region, _ := options.GetMetadata(utils.Region)
	if region == "" {
		region = "cn-hangzhou"
	}
	ecsClient, err := ecs.NewClientWithAccessKey(region, accessKey, secretKey)
	if err != nil {
		return nil, err
	}
	/*
		rmClient, err := resourcemanager.NewClientWithAccessKey(region, accessKey, secretKey)
		if err != nil {
			return nil, err
		}
		req := resourcemanager.CreateListResourceGroupsRequest()
		req.Scheme = "https"
		resp, err := rmClient.ListResourceGroups(req)
		if err != nil {
			return nil, err
		}
		var resourceGroups []string
		for _, group := range resp.ResourceGroups.ResourceGroup {
			resourceGroups = append(resourceGroups, group.Id)
		}
	*/
	resourceGroups := []string{""}

	return &Provider{
		vendor:         "alibaba",
		EcsClient:      ecsClient,
		resourceGroups: resourceGroups,
	}, nil
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return p.vendor
}

// Resources returns the provider for a resource deployment source.
func (p *Provider) Resources(ctx context.Context) (*schema.Resources, error) {
	list := schema.NewResources()
	list.Provider = p.vendor
	ecsprovider := &_ecs.InstanceProvider{Client: p.EcsClient, ResourceGroups: p.resourceGroups}
	list.Hosts, _ = ecsprovider.GetResource(ctx)

	return list, nil
}
