package huawei

import (
	"context"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/ecs"
	_iam "github.com/404tk/cloudtoolkit/pkg/providers/huawei/iam"
	_obs "github.com/404tk/cloudtoolkit/pkg/providers/huawei/obs"
	_rds "github.com/404tk/cloudtoolkit/pkg/providers/huawei/rds"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
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
		return nil, &schema.ErrNoSuchKey{Name: utils.AccessKey}
	}
	secretKey, ok := options.GetMetadata(utils.SecretKey)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: utils.SecretKey}
	}

	r := _iam.NewGetRequest()
	userName, err := r.GetUserName(accessKey, secretKey)
	if err != nil {
		return nil, err
	}
	log.Printf("[+] Current user: %s\n", userName)
	cache.Cfg.CredInsert(userName, options)

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
	payload, _ := options.GetMetadata(utils.Payload)
	var regions []string
	if regionId == "all" && payload == "cloudlist" {
		client := iam.NewIamClient(
			iam.IamClientBuilder().
				WithRegion(region.ValueOf("cn-east-2")).
				WithCredential(auth).
				Build())
		req := &model.KeystoneListRegionsRequest{}
		resp, err := client.KeystoneListRegions(req)
		if err != nil {
			log.Println("[-] List regions failed.")
			return nil, err
		}
		for _, r := range *resp.Regions {
			regions = append(regions, r.Id)
		}
	} else if regionId == "all" && payload != "cloudlist" {
		regions = append(regions, "cn-east-2")
	} else {
		regions = append(regions, regionId)
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
	var err error
	ecsprovider := &ecs.InstanceProvider{Auth: p.auth, Regions: p.regions}
	list.Hosts, err = ecsprovider.GetResource(ctx)

	obsprovider := &_obs.OBSProvider{Auth: p.auth, Regions: p.regions}
	list.Storages, err = obsprovider.GetBuckets(ctx)

	iamprovider := &_iam.IAMUserProvider{Auth: p.auth, Regions: p.regions}
	list.Users, err = iamprovider.GetIAMUser(ctx)

	rdsprovider := &_rds.RdsProvider{Auth: p.auth, Regions: p.regions}
	list.Databases, err = rdsprovider.GetDatabases(ctx)

	return list, err
}

func (p *Provider) UserManagement(action, uname, pwd string) {
	ramprovider := &_iam.IAMUserProvider{
		Auth: p.auth, Regions: p.regions, Username: uname, Password: pwd}
	switch action {
	case "add":
		ramprovider.AddUser()
	case "del":
		ramprovider.DelUser()
	default:
		log.Println("[-] Please set metadata like \"add username password\" or \"del username\"")
	}
}

func (p *Provider) BucketDump(action, bucketname string) {
	log.Println("[-] Not supported yet.")
}
