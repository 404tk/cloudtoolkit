package huawei

import (
	"context"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/ecs"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	iam "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3/model"
	region "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3/region"
)

// Provider is a data provider for huawei API
type Provider struct {
	vendor  string
	auth    basic.Credentials
	regions []string
}

// New creates a new provider client for huawei API
func New(options schema.OptionBlock) (*Provider, error) {
	accessKey, ok := options.GetMetadata(utils.AccessKey)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: accessKey}
	}
	secretKey, ok := options.GetMetadata(utils.SecretKey)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: secretKey}
	}
	auth := basic.NewCredentialsBuilder().
		WithAk(accessKey).
		WithSk(secretKey).
		Build()

	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
		}
	}()

	regionId, _ := options.GetMetadata(utils.Region)
	if regionId == "" {
		regionId = "cn-east-2"
	}
	client := iam.NewIamClient(
		iam.IamClientBuilder().
			WithRegion(region.ValueOf("cn-east-2")).
			WithCredential(auth).
			Build())

	req := &model.KeystoneListRegionsRequest{}
	resp, err := client.KeystoneListRegions(req)
	if err != nil {
		return nil, err
	}
	var regions []string
	for _, r := range *resp.Regions {
		regions = append(regions, r.Id)
	}

	return &Provider{
		vendor:  "huawei",
		auth:    auth,
		regions: regions,
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
	ecsprovider := &ecs.InstanceProvider{Auth: p.auth, Regions: p.regions}
	list.Hosts, _ = ecsprovider.GetResource(ctx)

	return list, nil
}
