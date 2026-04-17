package huawei

import (
	"context"

	_bss "github.com/404tk/cloudtoolkit/pkg/providers/huawei/bss"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/ecs"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/endpoint"
	_iam "github.com/404tk/cloudtoolkit/pkg/providers/huawei/iam"
	_obs "github.com/404tk/cloudtoolkit/pkg/providers/huawei/obs"
	_rds "github.com/404tk/cloudtoolkit/pkg/providers/huawei/rds"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	iam "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3/model"
)

// Provider is a data provider for huawei API
type Provider struct {
	auth    *basic.Credentials
	regions []string
	intl    bool
}

var defaultRegion = "cn-north-4"

// New creates a new provider client for huawei API
func New(options schema.Options) (*Provider, error) {
	accessKey, ok := options.GetMetadata(utils.AccessKey)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: utils.AccessKey}
	}
	secretKey, ok := options.GetMetadata(utils.SecretKey)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: utils.SecretKey}
	}
	regionId, _ := options.GetMetadata(utils.Region)
	v, _ := options.GetMetadata(utils.Version)
	intl := false
	if !(v == "" || v == "China") {
		intl = true
	}

	var r = &_iam.DefaultHttpRequest{}
	if regionId == "all" {
		r = _iam.NewGetRequest(defaultRegion)
	} else {
		r = _iam.NewGetRequest(regionId)
	}

	payload, _ := options.GetMetadata(utils.Payload)
	if payload == "cloudlist" {
		userName, err := r.GetUserName(accessKey, secretKey)
		if err != nil {
			return nil, err
		}
		msg := "Current user: " + userName
		cache.Cfg.CredInsert(userName, options)
		logger.Warning(msg)
	}

	auth := basic.NewCredentialsBuilder().
		WithAk(accessKey).
		WithSk(secretKey).
		Build()

	var regions []string
	if regionId == "all" && payload == "cloudlist" {
		client := iam.NewIamClient(
			iam.IamClientBuilder().
				WithEndpoint(endpoint.For("iam", defaultRegion, intl)).
				WithCredential(auth).
				Build())
		resp, err := client.KeystoneListRegions(&model.KeystoneListRegionsRequest{})
		if err != nil {
			logger.Error("List regions failed.")
			return nil, err
		}
		for _, r := range *resp.Regions {
			regions = append(regions, r.Id)
		}
	} else if regionId == "all" && payload != "cloudlist" {
		regions = append(regions, defaultRegion)
	} else {
		regions = append(regions, regionId)
	}

	return &Provider{
		auth:    auth,
		regions: regions,
		intl:    intl,
	}, nil
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return "huawei"
}

// Resources returns the provider for a resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	list := schema.NewResources()
	list.Provider = p.Name()
	for _, product := range utils.Cloudlist {
		switch product {
		case "balance":
			d := &_bss.Driver{Cred: p.auth, Intl: p.intl}
			d.QueryAccountBalance(ctx)
		case "host":
			ecsprovider := &ecs.Driver{Auth: p.auth, Regions: p.regions}
			hosts, err := ecsprovider.GetResource(ctx)
			schema.AppendAssets(&list, hosts)
			list.AddError("host", err)
		case "account":
			iamprovider := &_iam.Driver{Auth: p.auth}
			users, err := iamprovider.ListUsers(ctx)
			schema.AppendAssets(&list, users)
			list.AddError("account", err)
		case "database":
			rdsprovider := &_rds.Driver{Auth: p.auth, Regions: p.regions}
			databases, err := rdsprovider.GetDatabases(ctx)
			schema.AppendAssets(&list, databases)
			list.AddError("database", err)
		case "bucket":
			obsprovider := &_obs.Driver{Auth: p.auth, Regions: p.regions}
			storages, err := obsprovider.GetBuckets(ctx)
			schema.AppendAssets(&list, storages)
			list.AddError("bucket", err)
		default:
		}
	}

	return list, list.Err()
}

func (p *Provider) UserManagement(action, username, password string) {
	r := &_iam.Driver{
		Auth: p.auth, Username: username, Password: password}
	switch action {
	case "add":
		r.AddUser()
	case "del":
		r.DelUser()
	default:
		logger.Error("Please set metadata like \"add username password\" or \"del username\"")
	}
}
