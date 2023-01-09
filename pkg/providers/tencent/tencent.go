package tencent

import (
	"context"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/cam"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/cos"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/cvm"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/lighthouse"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
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
func New(options schema.OptionBlock) (*Provider, error) {
	accessKey, ok := options.GetMetadata(utils.AccessKey)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: utils.AccessKey}
	}
	secretKey, ok := options.GetMetadata(utils.SecretKey)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: utils.SecretKey}
	}

	credential := common.NewCredential(accessKey, secretKey)
	cpf := profile.NewClientProfile()
	region, _ := options.GetMetadata(utils.Region)

	request := sts.NewGetCallerIdentityRequest()
	// cpf.HttpProfile.Endpoint = "sts.tencentcloudapi.com"
	stsclient, _ := sts.NewClient(credential, "ap-guangzhou", cpf)
	response, err := stsclient.GetCallerIdentity(request)
	if err != nil {
		return nil, err
	}
	log.Printf("[+] Current account type: %s\n", *response.Response.Type)
	// accountId, _ := strconv.Atoi(*response.Response.UserId)
	cache.Cfg.CredInsert(*response.Response.Type, options)

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

	cosprovider := &cos.COSProvider{Credential: p.credential}
	list.Storages, err = cosprovider.GetBuckets(ctx)

	camprovider := &cam.CamUserProvider{Credential: p.credential}
	list.Users, err = camprovider.GetCamUser(ctx)

	return list, err
}

func (p *Provider) UserManagement(action, uname, pwd string) {
	ramprovider := &cam.CamUserProvider{
		Credential: p.credential, UserName: uname, Password: pwd}
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
	log.Println("[*] Recommended use https://cosbrowser.cloud.tencent.com/web")
}
