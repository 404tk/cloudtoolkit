package alibaba

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

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
	"github.com/404tk/cloudtoolkit/pkg/providers/internal/credverify"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/runtime/vmexecspec"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
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
	region = strings.TrimSpace(region)
	if strings.EqualFold(region, "all") {
		region = "all"
	}
	provider := &Provider{
		apiCred:          apiCred,
		region:           region,
		apiClientOptions: append([]_api.Option(nil), cfg.APIOptions...),
		ossClientOptions: append([]_oss.Option(nil), cfg.OSSOptions...),
		slsHTTPClient:    cfg.SLSHTTPClient,
	}

	if err := credverify.ForCloudlist(options, provider, cfg.SkipCredentialCache, func(ctx context.Context) (credverify.Result, error) {
		response, err := _api.NewClient(apiCred, cfg.APIOptions...).GetCallerIdentity(ctx, region)
		if err != nil {
			return credverify.Result{}, err
		}
		accountArn := response.Arn
		var userName string
		if len(accountArn) >= 4 && accountArn[len(accountArn)-4:] == "root" {
			userName = "root"
		} else if u := strings.Split(accountArn, "/"); len(u) > 1 {
			userName = u[1]
		}
		return credverify.Result{
			Summary:     fmt.Sprintf("Current user: %s (%s)", userName, accountArn),
			SessionUser: userName,
		}, nil
	}); err != nil {
		return nil, err
	}

	return provider, nil
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return "alibaba"
}

// Resources returns the provider for a resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	collector := schema.NewResourceCollector(p.Name()).
		Register("balance", func(ctx context.Context, _ *schema.Resources) {
			p.newBSSDriver(p.region).QueryAccountBalance(ctx)
		}).
		Register("host", func(ctx context.Context, list *schema.Resources) {
			ecsprovider := p.newECSDriver(p.region)
			hosts, err := ecsprovider.GetResource(ctx)
			schema.AppendAssets(list, hosts)
			list.AddError("host", err)
			list.AddError("host", ecsprovider.PartialError())
		}).
		Register("domain", func(ctx context.Context, list *schema.Resources) {
			dnsprovider := p.newDNSDriver(p.region)
			domains, err := dnsprovider.GetDomains(ctx)
			schema.AppendAssets(list, domains)
			list.AddError("domain", err)
		}).
		Register("account", func(ctx context.Context, list *schema.Resources) {
			ramprovider := p.newIAMDriver(p.region)
			users, err := ramprovider.ListUsers(ctx)
			schema.AppendAssets(list, users)
			list.AddError("account", err)
		}).
		Register("database", func(ctx context.Context, list *schema.Resources) {
			rdsprovider := p.newRDSDriver(p.region)
			databases, err := rdsprovider.GetDatabases(ctx)
			schema.AppendAssets(list, databases)
			list.AddError("database", err)
			list.AddError("database", rdsprovider.PartialError())
		}).
		Register("bucket", func(ctx context.Context, list *schema.Resources) {
			ossprovider := p.newOSSDriver(p.region)
			storages, err := ossprovider.GetBuckets(ctx)
			schema.AppendAssets(list, storages)
			list.AddError("bucket", err)
		}).
		Register("sms", func(ctx context.Context, list *schema.Resources) {
			smsprovider := p.newSMSDriver(p.region)
			sms, err := smsprovider.GetResource(ctx)
			list.Sms = sms
			list.AddError("sms", err)
		}).
		Register("log", func(ctx context.Context, list *schema.Resources) {
			slsprovider := p.newSLSDriver(p.region)
			logs, err := slsprovider.ListProjects(ctx)
			schema.AppendAssets(list, logs)
			list.AddError("log", err)
			list.AddError("log", slsprovider.PartialError())
		})

	return collector.Collect(ctx, env.From(ctx).Cloudlist)
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

// RoleBinding implements schema.RoleBindingManager for alibaba RAM. `principal`
// is a RAM user name, `role` is the policy name, and `scope` is the policy
// type (System or Custom; defaults to System).
func (p *Provider) RoleBinding(ctx context.Context, action, principal, role, scope string) (schema.RoleBindingResult, error) {
	driver := p.newIAMDriver(p.region)
	resolvedScope := scope
	if strings.TrimSpace(resolvedScope) == "" {
		resolvedScope = "System"
	}
	result := schema.RoleBindingResult{
		Action:    action,
		Principal: principal,
		Role:      role,
		Scope:     resolvedScope,
	}
	switch action {
	case "list":
		bindings, err := driver.ListRoleBindings(ctx, principal)
		if err != nil {
			return result, err
		}
		result.Bindings = bindings
		result.Message = fmt.Sprintf("%d policies attached to user %s", len(bindings), principal)
		return result, nil
	case "add":
		if err := driver.AttachPolicyToUser(ctx, principal, role, resolvedScope); err != nil {
			return result, err
		}
		result.Message = fmt.Sprintf("attached policy %s (%s) to user %s", role, resolvedScope, principal)
		return result, nil
	case "del":
		if err := driver.DetachPolicyFromUser(ctx, principal, role, resolvedScope); err != nil {
			return result, err
		}
		result.Message = fmt.Sprintf("detached policy %s (%s) from user %s", role, resolvedScope, principal)
		return result, nil
	}
	return result, fmt.Errorf("alibaba: unsupported role-binding action %q", action)
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

// BucketACL implements schema.BucketACLManager for alibaba OSS. `container`
// is a bucket name; `level` is the canned OSS ACL value (private,
// public-read, public-read-write) or a friendly alias resolved by
// NormalizeOSSACL.
func (p *Provider) BucketACL(ctx context.Context, action, container, level string) (schema.BucketACLResult, error) {
	driver := p.newOSSDriver(p.region)
	result := schema.BucketACLResult{
		Action:    action,
		Container: container,
		Level:     level,
	}
	switch action {
	case "audit":
		entries, err := driver.AuditBucketACL(ctx, container)
		if err != nil {
			return result, err
		}
		result.Containers = entries
		result.Message = fmt.Sprintf("%d buckets audited", len(entries))
		return result, nil
	case "expose":
		applied, err := driver.ExposeBucket(ctx, container, level)
		if err != nil {
			return result, err
		}
		result.Level = applied
		result.Message = fmt.Sprintf("bucket %s set to %s", container, applied)
		return result, nil
	case "unexpose":
		if err := driver.UnexposeBucket(ctx, container); err != nil {
			return result, err
		}
		result.Level = _oss.OSSACLPrivate
		result.Message = fmt.Sprintf("bucket %s reverted to private", container)
		return result, nil
	}
	return result, fmt.Errorf("alibaba: unsupported bucket-acl action %q", action)
}

func (p *Provider) EventDump(ctx context.Context, action, args string) (schema.EventActionResult, error) {
	d := p.newSASDriver()
	switch action {
	case "dump":
		events, err := d.DumpEvents(ctx)
		if err != nil {
			return schema.EventActionResult{}, err
		}
		return schema.EventActionResult{
			Action: "dump",
			Scope:  args,
			Events: events,
		}, nil
	case "whitelist":
		return d.HandleEvents(ctx, args)
	default:
		return schema.EventActionResult{}, fmt.Errorf("invalid action: %s (expected: dump, whitelist)", action)
	}
}

func (p *Provider) ExecuteCloudVMCommand(ctx context.Context, instanceID, cmd string) (schema.CommandResult, error) {
	if osType, command, ok := vmexecspec.Parse(cmd); ok {
		if p.region == "" || p.region == "all" {
			return schema.CommandResult{}, fmt.Errorf("headless shell requires explicit region")
		}
		output := p.newECSDriver(p.region).RunCommand(instanceID, osType, command)
		return schema.CommandResult{Output: output}, nil
	}

	host, ok := p.lookupHost(instanceID)
	if !ok {
		return schema.CommandResult{}, fmt.Errorf("unable to resolve instance metadata")
	}
	d := p.newECSDriver(host.Region)
	command, err := base64.StdEncoding.DecodeString(cmd)
	if err != nil {
		return schema.CommandResult{}, err
	}
	output := d.RunCommand(instanceID, host.OSType, string(command))
	return schema.CommandResult{Output: output}, nil
}

func (p *Provider) DBManagement(ctx context.Context, action, instanceID string) (schema.DatabaseActionResult, error) {
	r := p.newRDSDriver(p.region)
	switch action {
	case "useradd":
		db, ok := p.lookupDatabase(instanceID)
		if !ok {
			return schema.DatabaseActionResult{}, fmt.Errorf("unable to resolve database metadata, retry: shell <instance-id>")
		}
		r.Region = db.Region
		return r.CreateAccount(ctx, instanceID, db.DBNames)
	case "userdel":
		return r.DeleteAccount(ctx, instanceID)
	default:
		return schema.DatabaseActionResult{}, fmt.Errorf("`instanceId` is missing")
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
	case p.region != "" && p.region != "all":
		infos[bucketName] = p.region
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
