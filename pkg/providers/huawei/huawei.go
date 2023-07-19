package huawei

import (
	"context"
	"fmt"
	"log"

	_bss "github.com/404tk/cloudtoolkit/pkg/providers/huawei/bss"
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
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3/region"
)

// Provider is a data provider for huawei API
type Provider struct {
	vendor  string
	auth    basic.Credentials
	regions []string
}

var default_region = "cn-north-4"

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

	var r = &_iam.DefaultHttpRequest{}
	if regionId == "all" {
		r = _iam.NewGetRequest(default_region)
	} else {
		r = _iam.NewGetRequest(regionId)
	}

	userName, err := r.GetUserName(accessKey, secretKey)
	if err != nil {
		return nil, err
	}
	msg := "[+] Current user: " + userName
	cache.Cfg.CredInsert(userName, options)

	auth := basic.NewCredentialsBuilder().
		WithAk(accessKey).
		WithSk(secretKey).
		Build()

	amount, err := _bss.QueryBalance(auth)
	if err == nil {
		msg += fmt.Sprintf(", available cash amount: %v", amount)
	}
	log.Println(msg)

	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
		}
	}()

	payload, _ := options.GetMetadata(utils.Payload)
	var regions []string
	if regionId == "all" && payload == "cloudlist" {
		client := iam.NewIamClient(
			iam.IamClientBuilder().
				WithRegion(region.ValueOf(default_region)).
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
		regions = append(regions, default_region)
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
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	list := schema.NewResources()
	list.Provider = p.vendor
	var err error
	ecsprovider := &ecs.Driver{Auth: p.auth, Regions: p.regions}
	list.Hosts, err = ecsprovider.GetResource(ctx)

	obsprovider := &_obs.Driver{Auth: p.auth, Regions: p.regions}
	list.Storages, err = obsprovider.GetBuckets(ctx)

	iamprovider := &_iam.Driver{Auth: p.auth, Regions: p.regions}
	list.Users, err = iamprovider.GetIAMUser(ctx)

	rdsprovider := &_rds.Driver{Auth: p.auth, Regions: p.regions}
	list.Databases, err = rdsprovider.GetDatabases(ctx)

	return list, err
}

func (p *Provider) UserManagement(action, uname, pwd string) {
	r := &_iam.Driver{
		Auth: p.auth, Regions: p.regions, Username: uname, Password: pwd}
	switch action {
	case "add":
		r.AddUser()
	case "del":
		r.DelUser()
	default:
		log.Println("[-] Please set metadata like \"add username password\" or \"del username\"")
	}
}

func (p *Provider) BucketDump(ctx context.Context, action, bucketname string) {
	log.Println("[-] Not supported yet.")
}

func (p *Provider) EventDump(action, sourceIp string) {}
