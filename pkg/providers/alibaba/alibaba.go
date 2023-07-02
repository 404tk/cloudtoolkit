package alibaba

import (
	"context"
	"fmt"
	"log"
	"strings"

	_dns "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/dns"
	_ecs "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/ecs"
	_oss "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/oss"
	_ram "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/ram"
	_rds "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/rds"
	_sms "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/sms"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/bssopenapi"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/resourcemanager"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/sts"
)

// Provider is a data provider for alibaba API
type Provider struct {
	vendor         string
	cred           *credentials.StsTokenCredential
	region         string
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
	token, _ := options.GetMetadata(utils.SecurityToken)
	cred := credentials.NewStsTokenCredential(accessKey, secretKey, token)

	// Get current username
	stsclient, err := sts.NewClientWithOptions("cn-hangzhou", sdk.NewConfig(), cred)
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
	msg := "[+] Current user: " + userName
	cache.Cfg.CredInsert(userName, options)

	bssclient, _ := bssopenapi.NewClientWithOptions("cn-hangzhou", sdk.NewConfig(), cred)
	req_bss := bssopenapi.CreateQueryAccountBalanceRequest()
	req_bss.Scheme = "https"
	resp, err := bssclient.QueryAccountBalance(req_bss)
	if err == nil {
		if resp.Data.AvailableCashAmount != "" {
			msg += ", available cash amount: " + resp.Data.AvailableCashAmount
		}
	}
	log.Printf(msg)

	rmClient, err := resourcemanager.NewClientWithOptions("cn-hangzhou", sdk.NewConfig(), cred)
	if err != nil {
		return nil, err
	}
	req_rm := resourcemanager.CreateListResourceGroupsRequest()
	req_rm.Scheme = "https"
	resp_rm, err := rmClient.ListResourceGroups(req_rm)
	if err != nil {
		return nil, err
	}
	var resourceGroups []string
	for _, group := range resp_rm.ResourceGroups.ResourceGroup {
		resourceGroups = append(resourceGroups, group.Id)
	}
	log.Printf("[*] Found %d ResourceGroups", len(resourceGroups))
	if len(resourceGroups) == 0 {
		return nil, fmt.Errorf("ResourceGroup not found.")
	}
	return &Provider{
		vendor:         "alibaba",
		cred:           cred,
		region:         region,
		resourceGroups: resourceGroups,
	}, err
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
	ecsprovider := &_ecs.InstanceProvider{Cred: p.cred, Region: p.region, ResourceGroups: p.resourceGroups}
	list.Hosts, err = ecsprovider.GetResource(ctx)

	dnsprovider := &_dns.DnsProvider{Cred: p.cred, Region: p.region, ResourceGroups: p.resourceGroups}
	list.Domains, err = dnsprovider.GetDomains(ctx)

	ossprovider := &_oss.BucketProvider{Cred: p.cred, Region: p.region}
	list.Storages, err = ossprovider.GetBuckets(ctx)

	ramprovider := &_ram.RamProvider{Cred: p.cred, Region: p.region}
	list.Users, err = ramprovider.GetRamUser(ctx)

	rdsprovider := &_rds.RdsProvider{Cred: p.cred, Region: p.region, ResourceGroups: p.resourceGroups}
	list.Databases, err = rdsprovider.GetDatabases(ctx)

	smsprovider := &_sms.SmsProvider{Cred: p.cred, Region: p.region}
	list.Sms, err = smsprovider.GetResource(ctx)

	return list, err
}

func (p *Provider) UserManagement(action, args_1, args_2 string) {
	ramprovider := &_ram.RamProvider{Cred: p.cred, Region: p.region}
	switch action {
	case "add":
		ramprovider.UserName = args_1
		ramprovider.PassWord = args_2
		ramprovider.AddUser()
	case "del":
		ramprovider.UserName = args_1
		ramprovider.DelUser()
	case "shadow":
		ramprovider.RoleName = args_1
		ramprovider.AccountId = args_2
		ramprovider.AddRole()
	case "delrole":
		ramprovider.RoleName = args_1
		ramprovider.DelRole()
	default:
		log.Println("[-] Please set metadata like \"add username password\" or \"del username\"")
	}
}

func (p *Provider) BucketDump(action, bucketname string) {
	log.Println("[*] Recommended use https://github.com/aliyun/oss-browser")
}
