package alibaba

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	_api "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	_auth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
	_bss "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/bss"
	_dns "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/dns"
	_ecs "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/ecs"
	_iam "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/iam"
	_oss "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/oss"
	_rds "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/rds"
	_sas "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/sas"
	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/sls"
	_sms "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/sms"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/table"
)

// Provider is a data provider for alibaba API
type Provider struct {
	apiCred          _auth.Credential
	region           string
	apiClientOptions []_api.Option
	ossClientOptions []_oss.Option
	slsHTTPClient    *http.Client
}

// New creates a new provider client for alibaba API
func New(options schema.Options) (*Provider, error) {
	return NewWithConfig(options, ClientConfig{})
}

type ClientConfig struct {
	APIOptions          []_api.Option
	OSSOptions          []_oss.Option
	SLSHTTPClient       *http.Client
	SkipCredentialCache bool
}

// NewWithConfig creates a new provider client for alibaba API with injected
// transport options. This keeps payload behavior intact while allowing
// replay/test clients to flow through the real provider and driver stack.
func NewWithConfig(options schema.Options, cfg ClientConfig) (*Provider, error) {
	apiCred, err := _auth.FromOptions(options)
	if err != nil {
		return nil, err
	}
	region, _ := options.GetMetadata(utils.Region)
	provider := &Provider{
		apiCred:          apiCred,
		region:           region,
		apiClientOptions: append([]_api.Option(nil), cfg.APIOptions...),
		ossClientOptions: append([]_oss.Option(nil), cfg.OSSOptions...),
		slsHTTPClient:    cfg.SLSHTTPClient,
	}

	payload, _ := options.GetMetadata(utils.Payload)
	if payload == "cloudlist" {
		// Get current username
		response, err := _api.NewClient(apiCred, cfg.APIOptions...).GetCallerIdentity(context.Background(), region)
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
		if !cfg.SkipCredentialCache {
			cache.Cfg.CredInsert(userName, provider, options)
		}
		logger.Warning(msg)
	}

	return provider, nil
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return "alibaba"
}

// Resources returns the provider for a resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	list := schema.NewResources()
	list.Provider = p.Name()
	for _, product := range env.From(ctx).Cloudlist {
		switch product {
		case "balance":
			d := p.newBSSDriver(p.region)
			d.QueryAccountBalance(ctx)
		case "host":
			ecsprovider := p.newECSDriver(p.region)
			hosts, err := ecsprovider.GetResource(ctx)
			schema.AppendAssets(&list, hosts)
			list.AddError("host", err)
			list.AddError("host", ecsprovider.PartialError())
		case "domain":
			dnsprovider := p.newDNSDriver(p.region)
			domains, err := dnsprovider.GetDomains(ctx)
			schema.AppendAssets(&list, domains)
			list.AddError("domain", err)
		case "account":
			ramprovider := p.newIAMDriver(p.region)
			users, err := ramprovider.ListUsers(ctx)
			schema.AppendAssets(&list, users)
			list.AddError("account", err)
		case "database":
			rdsprovider := p.newRDSDriver(p.region)
			databases, err := rdsprovider.GetDatabases(ctx)
			schema.AppendAssets(&list, databases)
			list.AddError("database", err)
			list.AddError("database", rdsprovider.PartialError())
		case "bucket":
			ossprovider := p.newOSSDriver(p.region)
			storages, err := ossprovider.GetBuckets(ctx)
			schema.AppendAssets(&list, storages)
			list.AddError("bucket", err)
		case "sms":
			smsprovider := p.newSMSDriver(p.region)
			sms, err := smsprovider.GetResource(ctx)
			list.Sms = sms
			list.AddError("sms", err)
		case "log":
			slsprovider := p.newSLSDriver(p.region)
			logs, err := slsprovider.ListProjects(ctx)
			schema.AppendAssets(&list, logs)
			list.AddError("log", err)
			list.AddError("log", slsprovider.PartialError())
		default:
		}
	}

	return list, list.Err()
}

func (p *Provider) UserManagement(action, username, password string) (schema.IAMResult, error) {
	r := p.newIAMDriver(p.region)
	switch action {
	case "add":
		r.UserName = username
		r.Password = password
		return r.AddUser()
	case "del":
		r.UserName = username
		return r.DelUser()
	case "shadow":
		r.RoleName = username
		r.AccountId = password
		return r.AddRole()
	case "delrole":
		r.RoleName = username
		return r.DelRole()
	default:
		return schema.IAMResult{}, fmt.Errorf("invalid action: %s (expected: add, del, shadow, delrole)", action)
	}
}

func (p *Provider) BucketDump(ctx context.Context, action, bucketName string) ([]schema.BucketResult, error) {
	ossdrvier := p.newOSSDriver(p.region)
	switch action {
	case "list":
		infos, err := p.bucketInfos(context.Background(), ossdrvier, bucketName)
		if err != nil {
			return nil, fmt.Errorf("list buckets: %w", err)
		}
		return ossdrvier.ListObjects(ctx, infos)
	case "total":
		infos, err := p.bucketInfos(context.Background(), ossdrvier, bucketName)
		if err != nil {
			return nil, fmt.Errorf("list buckets: %w", err)
		}
		return ossdrvier.TotalObjects(ctx, infos)
	default:
		return nil, fmt.Errorf("invalid action: %s (expected: list, total)", action)
	}
}

func (p *Provider) EventDump(action, args string) {
	d := p.newSASDriver()
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
		e := env.Active()
		if e.LogEnable {
			filename := time.Now().Format("20060102150405.log")
			path := fmt.Sprintf("%s/%s_eventdump_%s", e.LogDir, p.Name(), filename)
			table.FileOutput(path, events)
			msg := fmt.Sprintf("Output written to [%s]", path)
			logger.Info(msg)
		}
	case "whitelist":
		d.HandleEvents(args) // args means SecurityEventIds
	default:
		logger.Error("Please set metadata like \"dump all\"")
	}
}

func (p *Provider) ExecuteCloudVMCommand(instanceID, cmd string) {
	host, ok := p.lookupHost(instanceID)
	if !ok {
		logger.Error("Unable to resolve instance metadata.")
		return
	}
	d := p.newECSDriver(host.Region)
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

func (p *Provider) DBManagement(action, instanceID string) {
	r := p.newRDSDriver(p.region)
	switch action {
	case "useradd":
		db, ok := p.lookupDatabase(instanceID)
		if !ok {
			logger.Error("Unable to resolve database metadata, retry: shell <instance-id>")
			return
		}
		r.Region = db.Region
		r.CreateAccount(instanceID, db.DBNames)
	case "userdel":
		r.DeleteAccount(instanceID)
	default:
		logger.Error("`instanceId` is missing")
	}
}

func (p *Provider) lookupHost(instanceID string) (schema.Host, bool) {
	for _, host := range _ecs.GetCacheHostList() {
		if host.ID == instanceID || host.HostName == instanceID {
			return host, true
		}
	}
	return schema.Host{}, false
}

func (p *Provider) lookupDatabase(instanceID string) (schema.Database, bool) {
	for _, db := range _rds.GetCacheDBList() {
		if db.InstanceId == instanceID {
			return db, true
		}
	}
	logger.Info("Database metadata cache miss, refreshing instances ...")
	driver := p.newRDSDriver(p.region)
	databases, err := driver.GetDatabases(context.Background())
	if err != nil {
		logger.Error(err)
		return schema.Database{}, false
	}
	for _, db := range databases {
		if db.InstanceId == instanceID {
			return db, true
		}
	}
	return schema.Database{}, false
}

func (p *Provider) bucketInfos(ctx context.Context, driver *_oss.Driver, bucketName string) (map[string]string, error) {
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

func (p *Provider) newBSSDriver(region string) *_bss.Driver {
	driver := &_bss.Driver{Cred: p.apiCred, Region: region}
	driver.SetClientOptions(p.apiClientOptions...)
	return driver
}

func (p *Provider) newDNSDriver(region string) *_dns.Driver {
	driver := &_dns.Driver{Cred: p.apiCred, Region: region}
	driver.SetClientOptions(p.apiClientOptions...)
	return driver
}

func (p *Provider) newECSDriver(region string) *_ecs.Driver {
	driver := &_ecs.Driver{Cred: p.apiCred, Region: region}
	driver.SetClientOptions(p.apiClientOptions...)
	return driver
}

func (p *Provider) newIAMDriver(region string) *_iam.Driver {
	driver := &_iam.Driver{Cred: p.apiCred, Region: region}
	driver.SetClientOptions(p.apiClientOptions...)
	return driver
}

func (p *Provider) newOSSDriver(region string) *_oss.Driver {
	driver := &_oss.Driver{Cred: p.apiCred, Region: region}
	if len(p.ossClientOptions) != 0 {
		driver.Client = _oss.NewClient(p.apiCred, p.ossClientOptions...)
	}
	return driver
}

func (p *Provider) newRDSDriver(region string) *_rds.Driver {
	driver := &_rds.Driver{Cred: p.apiCred, Region: region}
	driver.SetClientOptions(p.apiClientOptions...)
	return driver
}

func (p *Provider) newSASDriver() _sas.Driver {
	driver := _sas.Driver{Cred: p.apiCred}
	driver.SetClientOptions(p.apiClientOptions...)
	return driver
}

func (p *Provider) newSLSDriver(region string) *sls.Driver {
	driver := &sls.Driver{Cred: p.apiCred, Region: region}
	driver.SetHTTPClient(p.slsHTTPClient)
	return driver
}

func (p *Provider) newSMSDriver(region string) *_sms.Driver {
	driver := &_sms.Driver{Cred: p.apiCred, Region: region}
	driver.SetClientOptions(p.apiClientOptions...)
	return driver
}
