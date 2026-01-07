package alibaba

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	_bss "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/bss"
	_dns "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/dns"
	_ecs "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/ecs"
	_oss "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/oss"
	_iam "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/iam"
	_rds "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/rds"
	_sas "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/sas"
	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/sls"
	_sms "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/sms"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/table"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/sts"
)

// Provider is a data provider for alibaba API
type Provider struct {
	cred   *credentials.StsTokenCredential
	region string
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

	payload, _ := options.GetMetadata(utils.Payload)
	if payload == "cloudlist" {
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
		msg := fmt.Sprintf("Current user: %s (%s)", userName, accountArn)
		cache.Cfg.CredInsert(userName, options)
		logger.Warning(msg)
	}

	return &Provider{
		cred:   cred,
		region: region,
	}, nil
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return "alibaba"
}

// Resources returns the provider for a resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	list := schema.NewResources()
	list.Provider = p.Name()
	var err error
	for _, product := range utils.Cloudlist {
		switch product {
		case "balance":
			d := &_bss.Driver{Cred: p.cred, Region: p.region}
			d.QueryAccountBalance(ctx)
		case "host":
			ecsprovider := &_ecs.Driver{Cred: p.cred, Region: p.region}
			list.Hosts, err = ecsprovider.GetResource(ctx)
		case "domain":
			dnsprovider := &_dns.Driver{Cred: p.cred, Region: p.region}
			list.Domains, err = dnsprovider.GetDomains(ctx)
		case "account":
			ramprovider := &_iam.Driver{Cred: p.cred, Region: p.region}
			list.Users, err = ramprovider.ListUsers(ctx)
		case "database":
			rdsprovider := &_rds.Driver{Cred: p.cred, Region: p.region}
			list.Databases, err = rdsprovider.GetDatabases(ctx)
		case "bucket":
			ossprovider := &_oss.Driver{Cred: p.cred, Region: p.region}
			list.Storages, err = ossprovider.GetBuckets(ctx)
		case "sms":
			smsprovider := &_sms.Driver{Cred: p.cred, Region: p.region}
			list.Sms, err = smsprovider.GetResource(ctx)
		case "log":
			slsprovider := &sls.Driver{Cred: p.cred, Region: p.region}
			list.Logs, err = slsprovider.ListProjects(ctx)
		default:
		}
	}

	return list, err
}

func (p *Provider) UserManagement(action, username, password string) {
	r := &_iam.Driver{Cred: p.cred, Region: p.region}
	switch action {
	case "add":
		r.UserName = username
		r.Password = password
		r.AddUser()
	case "del":
		r.UserName = username
		r.DelUser()
	case "shadow":
		r.RoleName = username
		r.AccountId = password
		r.AddRole()
	case "delrole":
		r.RoleName = username
		r.DelRole()
	default:
		logger.Error("Please set metadata like \"add username password\" or \"del username\"")
	}
}

func (p *Provider) BucketDump(ctx context.Context, action, bucketName string) {
	ossdrvier := &_oss.Driver{Cred: p.cred, Region: p.region}
	switch action {
	case "list":
		var infos = make(map[string]string)
		if bucketName == "all" {
			buckets, _ := ossdrvier.GetBuckets(context.Background())
			for _, b := range buckets {
				infos[b.BucketName] = b.Region
			}
		} else {
			infos[bucketName] = p.region
		}
		ossdrvier.ListObjects(ctx, infos)
	case "total":
		var infos = make(map[string]string)
		if bucketName == "all" {
			buckets, _ := ossdrvier.GetBuckets(context.Background())
			for _, b := range buckets {
				infos[b.BucketName] = b.Region
			}
		} else {
			infos[bucketName] = p.region
		}
		ossdrvier.TotalObjects(ctx, infos)
	default:
		logger.Error("`list all` or `total all`.")
	}
}

func (p *Provider) EventDump(action, sourceIP string) {
	d := _sas.Driver{Cred: p.cred}
	switch action {
	case "dump":
		events, err := d.DumpEvents()
		if err != nil {
			logger.Error(err)
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
			msg := fmt.Sprintf("Output written to [%s]", path)
			logger.Info(msg)
		}
	case "whitelist":
		d.HandleEvents(sourceIP) // sourceIP here means SecurityEventIds
	default:
		logger.Error("Please set metadata like \"dump all\"")
	}
}

func (p *Provider) ExecuteCloudVMCommand(instanceID, cmd string) {
	var region, ostype string
	for _, host := range _ecs.GetCacheHostList() {
		if host.ID == instanceID {
			region = host.Region
			ostype = host.OSType
			break
		}
	}
	if region == "" {
		logger.Error("Run cloudlist first")
		return
	}
	d := _ecs.Driver{Cred: p.cred, Region: region}
	client, err := d.NewClient()
	if err != nil {
		logger.Error(err)
		return
	}
	command, err := base64.StdEncoding.DecodeString(cmd)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	output := _ecs.RunCommand(client, instanceID, region, ostype, string(command))
	if output != "" {
		fmt.Println(output)
	}
}

func (p *Provider) DBManagement(action, instanceID string) {
	r := &_rds.Driver{Cred: p.cred, Region: p.region}
	switch action {
	case "useradd":
		var region, dbname string
		//var instance schema.Database
		for _, db := range _rds.GetCacheDBList() {
			if db.InstanceId == instanceID {
				region = db.Region
				dbname = db.DBNames
				//instance = db
				break
			}
		}
		if region == "" {
			logger.Error("Run cloudlist first")
			return
		}
		r.Region = region
		r.CreateAccount(instanceID, dbname)
	case "userdel":
		r.DeleteAccount(instanceID)
	default:
		logger.Error("`instanceId` is missing")
	}
}
