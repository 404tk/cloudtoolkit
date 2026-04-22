package volcengine

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	_api "github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	_auth "github.com/404tk/cloudtoolkit/pkg/providers/volcengine/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/billing"
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/ecs"
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/iam"
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/rds"
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/tos"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Provider struct {
	credential _auth.Credential
	region     string
	apiClient  *_api.Client
}

// New creates a new provider client for volcengine API.
func New(options schema.Options) (*Provider, error) {
	return newProvider(options)
}

func newProvider(options schema.Options, clientOptions ..._api.Option) (*Provider, error) {
	credential, err := _auth.FromOptions(options)
	if err != nil {
		return nil, err
	}
	region, _ := options.GetMetadata(utils.Region)
	apiClient := _api.NewClient(credential, clientOptions...)

	payload, _ := options.GetMetadata(utils.Payload)
	if payload == "cloudlist" {
		name, err := (&iam.Driver{Client: apiClient, Region: region}).GetProject(context.Background())
		if err != nil {
			return nil, err
		}
		logger.Warning("Current project:", name)
		cache.Cfg.CredInsert(name, options)
	}

	return &Provider{
		credential: credential,
		region:     region,
		apiClient:  apiClient,
	}, nil
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return "volcengine"
}

// Resources returns the provider for a resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	list := schema.NewResources()
	list.Provider = p.Name()
	for _, product := range utils.Cloudlist {
		switch product {
		case "balance":
			(&billing.Driver{Client: p.apiClient, Region: p.region}).QueryAccountBalance(ctx)
		case "host":
			d := &ecs.Driver{Client: p.apiClient, Region: p.region}
			hosts, err := d.GetResource(ctx)
			schema.AppendAssets(&list, hosts)
			list.AddError("host", err)
		case "domain":
		case "account":
			d := &iam.Driver{Client: p.apiClient, Region: p.region}
			users, err := d.ListUsers(ctx)
			schema.AppendAssets(&list, users)
			list.AddError("account", err)
		case "database":
			d := &rds.Driver{Client: p.apiClient, Region: p.region}
			mysqls, err := d.ListMySQL(ctx)
			schema.AppendAssets(&list, mysqls)
			list.AddError("database/mysql", err)
			list.AddError("database/mysql", d.PartialError())
			postgres, err := d.ListPostgreSQL(ctx)
			schema.AppendAssets(&list, postgres)
			list.AddError("database/postgresql", err)
			list.AddError("database/postgresql", d.PartialError())
			mssqls, err := d.ListSQLServer(ctx)
			schema.AppendAssets(&list, mssqls)
			list.AddError("database/sqlserver", err)
			list.AddError("database/sqlserver", d.PartialError())
		case "bucket":
			d := p.newTOSDriver(p.region)
			storages, err := d.GetBuckets(ctx)
			schema.AppendAssets(&list, storages)
			list.AddError("bucket", err)
		case "sms":
		case "log":
		default:
		}
	}

	return list, list.Err()
}

func (p *Provider) UserManagement(action, username, password string) {
	driver := &iam.Driver{
		Client:   p.apiClient,
		Region:   p.region,
		UserName: username,
		Password: password,
	}

	switch action {
	case "add":
		driver.AddUser()
	case "del":
		driver.DelUser()
	default:
		logger.Error("Please set metadata like \"add username password\" or \"del username\"")
	}
}

func (p *Provider) BucketDump(ctx context.Context, action, bucketName string) {
	driver := p.newTOSDriver(p.region)
	switch action {
	case "list":
		infos, err := p.bucketInfos(context.Background(), driver, bucketName)
		if err != nil {
			logger.Error("List buckets failed:", err)
			return
		}
		driver.ListObjects(ctx, infos)
	case "total":
		infos, err := p.bucketInfos(context.Background(), driver, bucketName)
		if err != nil {
			logger.Error("List buckets failed:", err)
			return
		}
		driver.TotalObjects(ctx, infos)
	default:
		logger.Error("`list all` or `total all`.")
	}
}

func (p *Provider) ExecuteCloudVMCommand(instanceID, cmd string) {
	host, ok := p.lookupHost(instanceID)
	if !ok {
		logger.Error("Unable to resolve instance metadata, run `cloudlist` first and retry.")
		return
	}

	command, err := base64.StdEncoding.DecodeString(cmd)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	driver := &ecs.Driver{Client: p.apiClient, Region: host.Region}
	output := driver.RunCommand(instanceID, host.OSType, string(command))
	if output != "" {
		fmt.Println(output)
	}
}

func (p *Provider) lookupHost(instanceID string) (schema.Host, bool) {
	for _, host := range ecs.GetCacheHostList() {
		if host.ID == instanceID || host.HostName == instanceID {
			return host, true
		}
	}
	return schema.Host{}, false
}

func (p *Provider) bucketInfos(ctx context.Context, driver *tos.Driver, bucketName string) (map[string]string, error) {
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

func (p *Provider) newTOSDriver(region string) *tos.Driver {
	return &tos.Driver{
		Cred:   p.credential,
		Region: region,
	}
}
