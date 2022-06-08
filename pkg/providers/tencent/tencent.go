package tencent

import (
	"context"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/cvm"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/lighthouse"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

// Provider is a data provider for tencent API
type Provider struct {
	vendor     string
	credential *common.Credential
	cpf        *profile.ClientProfile
	region     string
}

// New creates a new provider client for tencent API
func New(options schema.OptionBlock) (*Provider, error) {
	accessKey, ok := options.GetMetadata(utils.AccessKey)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: accessKey}
	}
	secretKey, ok := options.GetMetadata(utils.SecretKey)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: secretKey}
	}

	credential := common.NewCredential(accessKey, secretKey)
	cpf := profile.NewClientProfile()
	region, _ := options.GetMetadata(utils.Region)

	return &Provider{
		vendor:     "tencent",
		credential: credential,
		cpf:        cpf,
		region:     region,
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
	cvmprovider := &cvm.InstanceProvider{
		Credential: p.credential, Cpf: p.cpf, Region: p.region}
	list.Hosts, _ = cvmprovider.GetResource(ctx)
	light := &lighthouse.InstanceProvider{
		Credential: p.credential, Cpf: p.cpf, Region: p.region}
	list.Hosts, _ = light.GetResource(ctx)

	return list, nil
}
