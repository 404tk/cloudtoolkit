package tencent

import (
	"context"
	"fmt"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/cam"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/cdb"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/cos"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/cvm"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/lighthouse"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	billing "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/billing/v20180709"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	sts "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sts/v20180813"
)

// Provider is a data provider for tencent API
type Provider struct {
	vendor     string
	credential *common.Credential
	region     string
}

// New creates a new provider client for tencent API
func New(options schema.Options) (*Provider, error) {
	accessKey, ok := options.GetMetadata(utils.AccessKey)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: utils.AccessKey}
	}
	secretKey, ok := options.GetMetadata(utils.SecretKey)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: utils.SecretKey}
	}
	token, _ := options.GetMetadata(utils.SecurityToken)
	region, _ := options.GetMetadata(utils.Region)

	credential := common.NewTokenCredential(accessKey, secretKey, token)
	cpf := profile.NewClientProfile()

	request := sts.NewGetCallerIdentityRequest()
	// cpf.HttpProfile.Endpoint = "sts.tencentcloudapi.com"
	stsclient, _ := sts.NewClient(credential, "ap-guangzhou", cpf)
	response, err := stsclient.GetCallerIdentity(request)
	if err != nil {
		return nil, err
	}
	msg := "[+] Current account type: " + *response.Response.Type
	// accountId, _ := strconv.Atoi(*response.Response.UserId)
	cache.Cfg.CredInsert(*response.Response.Type, options)

	// cpf.HttpProfile.Endpoint = "billing.tencentcloudapi.com"
	client, _ := billing.NewClient(credential, "ap-guangzhou", cpf)
	req_billing := billing.NewDescribeAccountBalanceRequest()
	resp_billing, err := client.DescribeAccountBalance(req_billing)
	if err == nil {
		cash := *resp_billing.Response.RealBalance / 100
		msg += fmt.Sprintf(", available cash amount: %v", cash)
	}
	log.Println(msg)

	return &Provider{
		vendor:     "tencent",
		credential: credential,
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
	var err error

	cvmprovider := &cvm.InstanceProvider{Credential: p.credential, Region: p.region}
	cvms, err := cvmprovider.GetResource(ctx)
	list.Hosts = append(list.Hosts, cvms...)

	light := &lighthouse.InstanceProvider{Credential: p.credential, Region: p.region}
	lights, err := light.GetResource(ctx)
	list.Hosts = append(list.Hosts, lights...)

	cdbprovider := cdb.CdbProvider{Credential: p.credential, Region: p.region}
	mysqls, err := cdbprovider.ListMySQL(ctx)
	list.Databases = append(list.Databases, mysqls...)
	mariadbs, err := cdbprovider.ListMariaDB(ctx)
	list.Databases = append(list.Databases, mariadbs...)
	postgres, err := cdbprovider.ListPostgreSQL(ctx)
	list.Databases = append(list.Databases, postgres...)
	mssqls, err := cdbprovider.ListSQLServer(ctx)
	list.Databases = append(list.Databases, mssqls...)

	cosprovider := &cos.COSProvider{Credential: p.credential}
	list.Storages, err = cosprovider.GetBuckets(ctx)

	camprovider := &cam.CamUserProvider{Credential: p.credential}
	list.Users, err = camprovider.GetCamUser(ctx)

	return list, err
}

func (p *Provider) UserManagement(action, args_1, args_2 string) {
	camprovider := &cam.CamUserProvider{Credential: p.credential}
	switch action {
	case "add":
		camprovider.UserName = args_1
		camprovider.Password = args_2
		camprovider.AddUser()
	case "del":
		camprovider.UserName = args_1
		camprovider.DelUser()
	case "shadow":
		camprovider.RoleName = args_1
		camprovider.Uin = args_2
		camprovider.AddRole()
	case "delrole":
		camprovider.RoleName = args_1
		camprovider.DelRole()
	default:
		log.Println("[-] Please set metadata like \"add username password\" or \"del username\"")
	}
}

func (p *Provider) BucketDump(action, bucketname string) {
	log.Println("[*] Recommended use https://cosbrowser.cloud.tencent.com/web")
}
