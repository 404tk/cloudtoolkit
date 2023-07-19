package alibaba

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	_dns "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/dns"
	_ecs "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/ecs"
	_oss "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/oss"
	_ram "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/ram"
	_rds "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/rds"
	_sas "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/sas"
	_sms "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/sms"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/cloudtoolkit/utils/table"
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

	return &Provider{
		vendor: "alibaba",
		cred:   cred,
		region: region,
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
	getResourceGroups(p)
	if len(p.resourceGroups) == 0 {
		return list, fmt.Errorf("ResourceGroup not found.")
	} else {
		log.Printf("[*] Found %d ResourceGroups\n", len(p.resourceGroups))
	}
	var err error
	ecsprovider := &_ecs.Driver{Cred: p.cred, Region: p.region, ResourceGroups: p.resourceGroups}
	list.Hosts, err = ecsprovider.GetResource(ctx)

	dnsprovider := &_dns.Driver{Cred: p.cred, Region: p.region, ResourceGroups: p.resourceGroups}
	list.Domains, err = dnsprovider.GetDomains(ctx)

	ossprovider := &_oss.Driver{Cred: p.cred, Region: p.region}
	list.Storages, err = ossprovider.GetBuckets(ctx)

	ramprovider := &_ram.Driver{Cred: p.cred, Region: p.region}
	list.Users, err = ramprovider.GetRamUser(ctx)

	rdsprovider := &_rds.Driver{Cred: p.cred, Region: p.region, ResourceGroups: p.resourceGroups}
	list.Databases, err = rdsprovider.GetDatabases(ctx)

	smsprovider := &_sms.Driver{Cred: p.cred, Region: p.region}
	list.Sms, err = smsprovider.GetResource(ctx)

	return list, err
}

func getResourceGroups(p *Provider) {
	rmClient, err := resourcemanager.NewClientWithOptions("cn-hangzhou", sdk.NewConfig(), p.cred)
	if err != nil {
		return
	}
	req_rm := resourcemanager.CreateListResourceGroupsRequest()
	req_rm.Scheme = "https"
	resp_rm, err := rmClient.ListResourceGroups(req_rm)
	if err != nil {
		return
	}
	for _, group := range resp_rm.ResourceGroups.ResourceGroup {
		p.resourceGroups = append(p.resourceGroups, group.Id)
	}
}

func (p *Provider) UserManagement(action, args_1, args_2 string) {
	r := &_ram.Driver{Cred: p.cred, Region: p.region}
	switch action {
	case "add":
		r.UserName = args_1
		r.PassWord = args_2
		r.AddUser()
	case "del":
		r.UserName = args_1
		r.DelUser()
	case "shadow":
		r.RoleName = args_1
		r.AccountId = args_2
		r.AddRole()
	case "delrole":
		r.RoleName = args_1
		r.DelRole()
	default:
		log.Println("[-] Please set metadata like \"add username password\" or \"del username\"")
	}
}

func (p *Provider) BucketDump(ctx context.Context, action, bucketname string) {
	ossdrvier := &_oss.Driver{Cred: p.cred, Region: p.region}
	switch action {
	case "list":
		var infos = make(map[string]string)
		if bucketname == "all" {
			buckets, _ := ossdrvier.GetBuckets(context.Background())
			for _, b := range buckets {
				infos[b.BucketName] = b.Region
			}
		} else {
			infos[bucketname] = p.region
		}
		ossdrvier.ListObjects(ctx, infos)
	case "total":
		var infos = make(map[string]string)
		if bucketname == "all" {
			buckets, _ := ossdrvier.GetBuckets(context.Background())
			for _, b := range buckets {
				infos[b.BucketName] = b.Region
			}
		} else {
			infos[bucketname] = p.region
		}
		ossdrvier.TotalObjects(ctx, infos)
	default:
		log.Println("[-] `list all` or `total all`.")
	}
}

func (p *Provider) EventDump(action, sourceIp string) {
	d := _sas.Driver{Cred: p.cred}
	switch action {
	case "dump":
		events, err := d.DumpEvents()
		if err != nil {
			log.Println("[-]", err)
			return
		}
		if len(events) == 0 {
			return
		}
		table.Output(events)
		if utils.DoSave {
			filename := time.Now().Format("20060102150405.log")
			path := fmt.Sprintf("%s/%s_eventdump_%s", utils.LogDir, p.Name(), filename)
			table.FileOutput(path, events)
			log.Printf("[+] Output written to [%s]\n", path)
		}
	case "whitelist":
		d.HandleEvents(sourceIp) // sourceIp here means SecurityEventIds
	default:
		log.Println("[-] Please set metadata like \"dump all\"")
	}
}
