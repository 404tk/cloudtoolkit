package tencent

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/billing"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/cdb"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/cos"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/cvm"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/dns"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/iam"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/lighthouse"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/tat"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	sts "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sts/v20180813"
)

// Provider is a data provider for tencent API
type Provider struct {
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

	payload, _ := options.GetMetadata(utils.Payload)
	if payload == "cloudlist" {
		request := sts.NewGetCallerIdentityRequest()
		// cpf.HttpProfile.Endpoint = "sts.tencentcloudapi.com"
		stsclient, err := sts.NewClient(credential, "ap-guangzhou", cpf)
		if err != nil {
			return nil, err
		}
		response, err := stsclient.GetCallerIdentity(request)
		if err != nil {
			return nil, err
		}
		msg := "Current account ARN: " + *response.Response.Arn
		// accountId, _ := strconv.Atoi(*response.Response.UserId)
		cache.Cfg.CredInsert(*response.Response.Type, options)
		logger.Warning(msg)
	}

	return &Provider{
		credential: credential,
		region:     region,
	}, nil
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return "tencent"
}

// Resources returns the provider for a resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	list := schema.NewResources()
	list.Provider = p.Name()
	for _, product := range utils.Cloudlist {
		switch product {
		case "balance":
			d := &billing.Driver{Cred: p.credential, Region: p.region}
			d.QueryAccountBalance(ctx)
		case "host":
			cvmprovider := &cvm.Driver{Credential: p.credential, Region: p.region}
			cvms, err := cvmprovider.GetResource(ctx)
			list.Hosts = append(list.Hosts, cvms...)
			list.AddError("host/cvm", err)
			light := &lighthouse.Driver{Credential: p.credential, Region: p.region}
			lights, err := light.GetResource(ctx)
			list.Hosts = append(list.Hosts, lights...)
			list.AddError("host/lighthouse", err)
			tat.SetCacheHostList(list.Hosts)
		case "domain":
			dnsprovider := &dns.Driver{Credential: p.credential}
			domains, err := dnsprovider.GetDomains(ctx)
			list.Domains = append(list.Domains, domains...)
			list.AddError("domain", err)
		case "account":
			camprovider := &iam.Driver{Credential: p.credential}
			users, err := camprovider.ListUsers(ctx)
			list.Users = append(list.Users, users...)
			list.AddError("account", err)
		case "database":
			cdbprovider := cdb.Driver{Credential: p.credential, Region: p.region}
			mysqls, err := cdbprovider.ListMySQL(ctx)
			list.Databases = append(list.Databases, mysqls...)
			list.AddError("database/mysql", err)
			mariadbs, err := cdbprovider.ListMariaDB(ctx)
			list.Databases = append(list.Databases, mariadbs...)
			list.AddError("database/mariadb", err)
			postgres, err := cdbprovider.ListPostgreSQL(ctx)
			list.Databases = append(list.Databases, postgres...)
			list.AddError("database/postgresql", err)
			mssqls, err := cdbprovider.ListSQLServer(ctx)
			list.Databases = append(list.Databases, mssqls...)
			list.AddError("database/sqlserver", err)
		case "bucket":
			cosprovider := &cos.Driver{Credential: p.credential}
			storages, err := cosprovider.GetBuckets(ctx)
			list.Storages = append(list.Storages, storages...)
			list.AddError("bucket", err)
		default:
		}
	}

	return list, list.Err()
}

func (p *Provider) UserManagement(action, username, password string) {
	c := &iam.Driver{Credential: p.credential}
	switch action {
	case "add":
		c.UserName = username
		c.Password = password
		c.AddUser()
	case "del":
		c.UserName = username
		c.DelUser()
	case "shadow":
		c.RoleName = username
		c.Uin = password
		c.AddRole()
	case "delrole":
		c.RoleName = username
		c.DelRole()
	default:
		logger.Error("Please set metadata like \"add username password\" or \"del username\"")
	}
}

func (p *Provider) BucketDump(ctx context.Context, action, bucketName string) {
	logger.Info("Recommended use https://cosbrowser.cloud.tencent.com/web")
}

func (p *Provider) EventDump(action, args string) {}

func (p *Provider) ExecuteCloudVMCommand(instanceID, cmd string) {
	host, ok := p.lookupHost(instanceID)
	if !ok {
		logger.Error("Unable to resolve instance metadata.")
		return
	}
	d := tat.Driver{Credential: p.credential, Region: host.Region}
	command, err := base64.StdEncoding.DecodeString(cmd)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	output := d.RunCommand(instanceID, host.OSType, string(command))
	if output != "" {
		fmt.Println(output)
	}
}

func (p *Provider) DBManagement(action, instanceID string) {}

func (p *Provider) lookupHost(instanceID string) (schema.Host, bool) {
	for _, host := range tat.GetCacheHostList() {
		if host.ID == instanceID {
			return host, true
		}
	}
	logger.Info("Host metadata cache miss, refreshing instances ...")
	cvmProvider := &cvm.Driver{Credential: p.credential, Region: p.region}
	hosts, err := cvmProvider.GetResource(context.Background())
	if err != nil {
		logger.Error(err)
	}
	lightProvider := &lighthouse.Driver{Credential: p.credential, Region: p.region}
	lights, lightErr := lightProvider.GetResource(context.Background())
	if lightErr != nil {
		logger.Error(lightErr)
	}
	hosts = append(hosts, lights...)
	if len(hosts) > 0 {
		tat.SetCacheHostList(hosts)
	}
	for _, host := range hosts {
		if host.ID == instanceID {
			return host, true
		}
	}
	return schema.Host{}, false
}
