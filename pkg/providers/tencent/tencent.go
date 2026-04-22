package tencent

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
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
)

// Provider is a data provider for tencent API
type Provider struct {
	apiCredential auth.Credential
	apiClient     *api.Client
	clientOptions []api.Option
	region        string
}

// New creates a new provider client for tencent API
func New(options schema.Options) (*Provider, error) {
	return newProvider(options)
}

func newProvider(options schema.Options, clientOptions ...api.Option) (*Provider, error) {
	credential, err := auth.FromOptions(options)
	if err != nil {
		return nil, err
	}
	region, _ := options.GetMetadata(utils.Region)
	opts := append([]api.Option(nil), clientOptions...)
	apiClient := api.NewClient(credential, opts...)

	payload, _ := options.GetMetadata(utils.Payload)
	if payload == "cloudlist" {
		var response api.GetCallerIdentityResponse
		err := apiClient.DoJSON(
			context.Background(),
			"sts",
			"2018-08-13",
			"GetCallerIdentity",
			"ap-guangzhou",
			api.GetCallerIdentityRequest{},
			&response,
		)
		if err != nil {
			return nil, err
		}
		msg := "Current account ARN: " + response.Response.Arn
		cache.Cfg.CredInsert(response.Response.Type, options)
		logger.Warning(msg)
	}

	return &Provider{
		apiCredential: credential,
		apiClient:     apiClient,
		clientOptions: opts,
		region:        region,
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
			d := &billing.Driver{Cred: p.apiCredential, Region: p.region}
			d.SetClientOptions(p.clientOptions...)
			d.QueryAccountBalance(ctx)
		case "host":
			cvmprovider := &cvm.Driver{Credential: p.apiCredential, Region: p.region}
			cvmprovider.SetClientOptions(p.clientOptions...)
			cvms, err := cvmprovider.GetResource(ctx)
			schema.AppendAssets(&list, cvms)
			list.AddError("host/cvm", err)
			list.AddError("host/cvm", cvmprovider.PartialError())
			light := &lighthouse.Driver{Credential: p.apiCredential, Region: p.region}
			light.SetClientOptions(p.clientOptions...)
			lights, err := light.GetResource(ctx)
			schema.AppendAssets(&list, lights)
			list.AddError("host/lighthouse", err)
			list.AddError("host/lighthouse", light.PartialError())
			allHosts := append(cvms, lights...)
			tat.SetCacheHostList(allHosts)
		case "domain":
			dnsprovider := &dns.Driver{Credential: p.apiCredential, Region: p.region}
			dnsprovider.SetClientOptions(p.clientOptions...)
			domains, err := dnsprovider.GetDomains(ctx)
			schema.AppendAssets(&list, domains)
			list.AddError("domain", err)
		case "account":
			camprovider := &iam.Driver{Credential: p.apiCredential}
			camprovider.SetClientOptions(p.clientOptions...)
			users, err := camprovider.ListUsers(ctx)
			schema.AppendAssets(&list, users)
			list.AddError("account", err)
		case "database":
			cdbprovider := &cdb.Driver{Credential: p.apiCredential, Region: p.region}
			cdbprovider.SetClientOptions(p.clientOptions...)
			mysqls, err := cdbprovider.ListMySQL(ctx)
			schema.AppendAssets(&list, mysqls)
			list.AddError("database/mysql", err)
			list.AddError("database/mysql", cdbprovider.PartialError())
			mariadbs, err := cdbprovider.ListMariaDB(ctx)
			schema.AppendAssets(&list, mariadbs)
			list.AddError("database/mariadb", err)
			list.AddError("database/mariadb", cdbprovider.PartialError())
			postgres, err := cdbprovider.ListPostgreSQL(ctx)
			schema.AppendAssets(&list, postgres)
			list.AddError("database/postgresql", err)
			list.AddError("database/postgresql", cdbprovider.PartialError())
			mssqls, err := cdbprovider.ListSQLServer(ctx)
			schema.AppendAssets(&list, mssqls)
			list.AddError("database/sqlserver", err)
			list.AddError("database/sqlserver", cdbprovider.PartialError())
		case "bucket":
			cosprovider := &cos.Driver{Credential: p.apiCredential}
			storages, err := cosprovider.GetBuckets(ctx)
			schema.AppendAssets(&list, storages)
			list.AddError("bucket", err)
		default:
		}
	}

	return list, list.Err()
}

func (p *Provider) UserManagement(action, username, password string) {
	c := &iam.Driver{Credential: p.apiCredential}
	c.SetClientOptions(p.clientOptions...)
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
	cosprovider := &cos.Driver{Credential: p.apiCredential}
	switch action {
	case "list":
		infos, err := p.bucketInfos(context.Background(), cosprovider, bucketName)
		if err != nil {
			logger.Error("List buckets failed:", err)
			return
		}
		cosprovider.ListObjects(ctx, infos)
	case "total":
		infos, err := p.bucketInfos(context.Background(), cosprovider, bucketName)
		if err != nil {
			logger.Error("List buckets failed:", err)
			return
		}
		cosprovider.TotalObjects(ctx, infos)
	default:
		logger.Error("`list all` or `total all`.")
	}
}

func (p *Provider) ExecuteCloudVMCommand(instanceID, cmd string) {
	host, ok := p.lookupHost(instanceID)
	if !ok {
		logger.Error("Unable to resolve instance metadata, retry: shell <instance-id>")
		return
	}
	d := tat.Driver{Credential: p.apiCredential, Region: host.Region}
	d.SetClientOptions(p.clientOptions...)
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

func (p *Provider) lookupHost(instanceID string) (schema.Host, bool) {
	for _, host := range tat.GetCacheHostList() {
		if host.ID == instanceID || host.HostName == instanceID {
			return host, true
		}
	}
	return schema.Host{}, false
}

func (p *Provider) bucketInfos(ctx context.Context, driver *cos.Driver, bucketName string) (map[string]string, error) {
	infos := make(map[string]string)
	bucketName = strings.TrimSpace(bucketName)
	region := strings.TrimSpace(p.region)
	switch {
	case bucketName == "":
		return nil, fmt.Errorf("empty bucket name")
	case bucketName == "all":
		buckets, err := driver.GetBuckets(ctx)
		if err != nil {
			return nil, err
		}
		for _, bucket := range buckets {
			infos[bucket.BucketName] = bucket.Region
		}
		if len(infos) == 0 {
			return nil, fmt.Errorf("no buckets found")
		}
		return infos, nil
	case region != "" && region != "all":
		infos[bucketName] = region
		return infos, nil
	default:
		buckets, err := driver.GetBuckets(ctx)
		if err != nil {
			return nil, err
		}
		for _, bucket := range buckets {
			if bucket.BucketName == bucketName {
				infos[bucket.BucketName] = bucket.Region
				return infos, nil
			}
		}
		return nil, fmt.Errorf("bucket %s region not found; set region explicitly or use `list all` first", bucketName)
	}
}
