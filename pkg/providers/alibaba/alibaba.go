package alibaba

import (
	"context"
	"log"
	"strings"

	_ecs "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/ecs"
	_oss "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/oss"
	_ram "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/ram"
	_rds "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/rds"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ram"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/rds"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/sts"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// Provider is a data provider for alibaba API
type Provider struct {
	vendor         string
	EcsClient      *ecs.Client
	OssClient      *oss.Client
	RamClient      *ram.Client
	RdsClient      *rds.Client
	resourceGroups []string
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
	if region == "all" {
		region = "cn-hangzhou"
	}
	token, _ := options.GetMetadata(utils.SecurityToken)

	// Get current username
	stsclient, err := sts.NewClientWithStsToken(region, accessKey, secretKey, token)
	request := sts.CreateGetCallerIdentityRequest()
	request.Scheme = "https"
	response, err := stsclient.GetCallerIdentity(request)
	if err != nil {
		return nil, err
	}
	accountArn := response.Arn
	var userName string
	if len(accountArn) >= 4 && accountArn[len(accountArn)-4:] == "root" {
		userName = "root"
	} else {
		if u := strings.Split(accountArn, "/"); len(u) > 1 {
			userName = u[1]
		}
	}
	log.Printf("[+] Current user: %s\n", userName)
	cache.Cfg.CredInsert(userName, options)

	ecsClient, err := ecs.NewClientWithStsToken(region, accessKey, secretKey, token)
	ossClient, err := oss.New("oss-"+region+".aliyuncs.com", accessKey, secretKey, oss.SecurityToken(token))
	ramClient, err := ram.NewClientWithStsToken(region, accessKey, secretKey, token)
	rdsClient, err := rds.NewClientWithStsToken(region, accessKey, secretKey, token)
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
	return &Provider{
		vendor:         "alibaba",
		EcsClient:      ecsClient,
		OssClient:      ossClient,
		RamClient:      ramClient,
		RdsClient:      rdsClient,
		resourceGroups: []string{""},
	}, err
}

/*
func GetCredInfo(region, accessKey, secretKey, token string) bool {

}
*/
// Name returns the name of the provider
func (p *Provider) Name() string {
	return p.vendor
}

// Resources returns the provider for a resource deployment source.
func (p *Provider) Resources(ctx context.Context) (*schema.Resources, error) {
	list := schema.NewResources()
	list.Provider = p.vendor
	var err error
	ecsprovider := &_ecs.InstanceProvider{Client: p.EcsClient, ResourceGroups: p.resourceGroups}
	list.Hosts, err = ecsprovider.GetResource(ctx)

	ossprovider := &_oss.BucketProvider{Client: p.OssClient}
	list.Storages, err = ossprovider.GetBuckets(ctx)

	ramprovider := &_ram.RamProvider{Client: p.RamClient}
	list.Users, err = ramprovider.GetRamUser(ctx)

	rdsprovider := &_rds.RdsProvider{Client: p.RdsClient, ResourceGroups: p.resourceGroups}
	list.Databases, err = rdsprovider.GetDatabases(ctx)

	return list, err
}

func (p *Provider) UserManagement(action, uname, pwd string) {
	ramprovider := &_ram.RamProvider{
		Client: p.RamClient, UserName: uname, PassWord: pwd}
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
	log.Println("[*] Recommended use https://github.com/aliyun/oss-browser")
}
