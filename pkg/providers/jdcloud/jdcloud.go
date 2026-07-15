package jdcloud

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	awsapi "github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/internal/credverify"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/actiontrail"
	_api "github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/asset"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/assistant"
	_auth "github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/dns"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/iam"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/lavm"
	jdlogs "github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/logs"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/oss"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/rds"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/vm"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/runtime/vmexecspec"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
)

type Provider struct {
	credential       _auth.Credential
	region           string
	accessKey        string
	apiClient        *_api.Client
	apiOptions       []_api.Option
	objectAPIOptions []awsapi.Option
}

// ClientConfig allows callers (e.g. demo replay) to inject custom api.Option
// values and skip credential cache writes for ephemeral credentials.
type ClientConfig struct {
	APIOptions          []_api.Option
	ObjectAPIOptions    []awsapi.Option
	SkipCredentialCache bool
}

// New creates a new provider client for JDCloud API.
func New(options schema.Options) (*Provider, error) {
	return NewWithConfig(options, ClientConfig{})
}

// NewWithConfig creates a new provider client for JDCloud API with injected
// transport options. Real callers use New; replay/test callers feed in a
// mock HTTP client through cfg.APIOptions.
func NewWithConfig(options schema.Options, cfg ClientConfig) (*Provider, error) {
	credential, err := _auth.FromOptions(options)
	if err != nil {
		return nil, err
	}
	region, _ := options.GetMetadata(utils.Region)
	region = strings.TrimSpace(region)
	if strings.EqualFold(region, "all") {
		region = "all"
	}
	apiClient := _api.NewClient(credential, cfg.APIOptions...)
	provider := &Provider{
		credential:       credential,
		region:           region,
		accessKey:        credential.AccessKey,
		apiClient:        apiClient,
		apiOptions:       append([]_api.Option(nil), cfg.APIOptions...),
		objectAPIOptions: append([]awsapi.Option(nil), cfg.ObjectAPIOptions...),
	}
	if err := credverify.ForCloudlist(options, provider, cfg.SkipCredentialCache, func(context.Context) (credverify.Result, error) {
		d := &iam.Driver{Client: apiClient, AccessKey: credential.AccessKey}
		pin, ok := d.Validator()
		if !ok {
			return credverify.Result{}, fmt.Errorf("invalid accesskey")
		}
		sessionUser := pin
		if sessionUser == "" {
			sessionUser = "default"
		}
		summary := ""
		if pin != "" {
			summary = fmt.Sprintf("Current user: %s", pin)
		}
		return credverify.Result{
			Summary:     summary,
			SessionUser: sessionUser,
		}, nil
	}); err != nil {
		return nil, err
	}

	return provider, nil
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return "jdcloud"
}

// Resources returns the provider for a resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	collector := schema.NewResourceCollector(p.Name()).
		Register("balance", func(ctx context.Context, _ *schema.Resources) {
			(&asset.Driver{Client: p.apiClient, Region: p.region}).QueryAccountBalance(ctx)
		}).
		Register("host", func(ctx context.Context, list *schema.Resources) {
			vmDriver := &vm.Driver{Client: p.apiClient, Region: p.region}
			vmHosts, vmErr := vmDriver.GetResource(ctx)
			schema.AppendAssets(list, vmHosts)
			list.AddError("host/vm", vmErr)

			lavmDriver := &lavm.Driver{Client: p.apiClient, Region: p.region}
			lavmHosts, lavmErr := lavmDriver.GetResource(ctx)
			schema.AppendAssets(list, lavmHosts)
			list.AddError("host/lavm", lavmErr)

			allHosts := append([]schema.Host{}, vmHosts...)
			allHosts = append(allHosts, lavmHosts...)
			if len(allHosts) > 0 || (vmErr == nil && lavmErr == nil) {
				vm.SetCacheHostList(allHosts)
			}
		}).
		Register("account", func(ctx context.Context, list *schema.Resources) {
			d := &iam.Driver{Client: p.apiClient}
			users, err := d.ListUsers(ctx)
			schema.AppendAssets(list, users)
			list.AddError("account", err)
		}).
		Register("bucket", func(ctx context.Context, list *schema.Resources) {
			d := p.newOSSDriver(p.region)
			storages, err := d.ListBuckets(ctx)
			schema.AppendAssets(list, storages)
			list.AddError("bucket", err)
		}).
		Register("log", func(ctx context.Context, list *schema.Resources) {
			d := &jdlogs.Driver{Client: p.apiClient, Region: p.region}
			logs, err := d.GetLogs(ctx)
			schema.AppendAssets(list, logs)
			list.AddError("log", err)
		}).
		Register("database", func(ctx context.Context, list *schema.Resources) {
			d := &rds.Driver{Client: p.apiClient, Region: p.region}
			dbs, err := d.GetDatabases(ctx)
			schema.AppendAssets(list, dbs)
			list.AddError("database", err)
		}).
		Register("domain", func(ctx context.Context, list *schema.Resources) {
			d := &dns.Driver{Client: p.apiClient, Region: p.region}
			domains, err := d.GetDomains(ctx)
			schema.AppendAssets(list, domains)
			list.AddError("domain", err)
		})

	return collector.Collect(ctx, env.From(ctx).Cloudlist)
}

// UserManagement powers the iam-user-check payload. JDCloud's CreateSubUser is
// atomic (name + password + consoleLogin in one call), so we only need an
// AttachSubUserPolicy follow-up to grant administrator privilege.
func (p *Provider) UserManagement(action, username, password string) (schema.IAMResult, error) {
	driver := &iam.Driver{
		Client:    p.apiClient,
		AccessKey: p.accessKey,
		UserName:  username,
		Password:  password,
	}
	switch action {
	case "add":
		return driver.AddUser()
	case "del":
		return driver.DelUser()
	default:
		return schema.IAMResult{}, fmt.Errorf("invalid action: %s (expected: add, del)", action)
	}
}

// RoleBinding implements schema.RoleBindingManager for JDCloud IAM. `principal`
// is the sub user name, `role` is the policy name (e.g. `JDCloudAdmin-New`).
// `scope` is reserved (JDCloud sub-user policies are not scoped per resource).
func (p *Provider) RoleBinding(ctx context.Context, action, principal, role, scope string) (schema.RoleBindingResult, error) {
	driver := &iam.Driver{Client: p.apiClient, AccessKey: p.accessKey}
	result := schema.RoleBindingResult{
		Action:    action,
		Principal: principal,
		Role:      role,
		Scope:     scope,
	}
	switch action {
	case "list":
		bindings, err := driver.ListRoleBindings(ctx, principal)
		if err != nil {
			return result, err
		}
		result.Bindings = bindings
		result.Message = fmt.Sprintf("%d policies attached to sub user %s", len(bindings), principal)
		return result, nil
	case "add":
		if err := driver.AttachPolicy(ctx, principal, role); err != nil {
			return result, err
		}
		result.Message = fmt.Sprintf("attached policy %s to sub user %s", role, principal)
		return result, nil
	case "del":
		if err := driver.DetachPolicy(ctx, principal, role); err != nil {
			return result, err
		}
		result.Message = fmt.Sprintf("detached policy %s from sub user %s", role, principal)
		return result, nil
	}
	return result, fmt.Errorf("jdcloud: unsupported role-binding action %q", action)
}

func (p *Provider) BucketDump(ctx context.Context, action, bucketName string) ([]schema.BucketResult, error) {
	driver := p.newOSSDriver(p.region)
	infos, err := p.bucketInfos(context.Background(), driver, bucketName)
	if err != nil {
		return nil, fmt.Errorf("list buckets: %w", err)
	}

	switch action {
	case "list":
		return driver.ListObjects(ctx, infos)
	case "total":
		return driver.TotalObjects(ctx, infos)
	default:
		return nil, fmt.Errorf("invalid action: %s (expected: list, total)", action)
	}
}

// IAMCredential implements schema.IAMCredentialManager for JDCloud IAM. The
// access-key lifecycle follows the same sub-user action family as
// `:describeAttachedPolicies` and `:attachSubUserPolicy`.
func (p *Provider) IAMCredential(ctx context.Context, action, principal, credentialID string) (schema.IAMCredentialResult, error) {
	driver := &iam.Driver{Client: p.apiClient, AccessKey: p.accessKey}
	result := schema.IAMCredentialResult{
		Action:       action,
		Principal:    principal,
		CredentialID: credentialID,
	}
	switch action {
	case "list":
		creds, err := driver.ListAccessKeys(ctx, principal)
		if err != nil {
			return result, err
		}
		result.Credentials = creds
		result.Message = fmt.Sprintf("%d access keys on %s", len(creds), principal)
		return result, nil
	case "create":
		cred, secret, err := driver.CreateAccessKey(ctx, principal)
		if err != nil {
			return result, err
		}
		result.CredentialID = cred.CredentialID
		result.CredentialData = secret
		result.Message = fmt.Sprintf("minted access key %s for %s", cred.CredentialID, principal)
		return result, nil
	case "delete":
		if err := driver.DeleteAccessKey(ctx, principal, credentialID); err != nil {
			return result, err
		}
		result.Message = fmt.Sprintf("revoked access key %s on %s", credentialID, principal)
		return result, nil
	}
	return result, fmt.Errorf("jdcloud: unsupported iam-credential action %q", action)
}

// BucketACL implements schema.BucketACLManager for JDCloud OSS (S3-compatible
// data plane). `level` accepts the canned S3-style ACL values (private,
// public-read, public-read-write, authenticated-read) or friendly aliases
// resolved by oss.NormalizeOSSACL.
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
		result.Level = oss.OSSACLPrivate
		result.Message = fmt.Sprintf("bucket %s reverted to private", container)
		return result, nil
	}
	return result, fmt.Errorf("jdcloud: unsupported bucket-acl action %q", action)
}

// EventDump implements schema.EventReader for JDCloud AuditTrail. `dump`
// lists recent audit events; `whitelist` is unsupported because AuditTrail is
// read-only.
func (p *Provider) EventDump(ctx context.Context, action, args string) (schema.EventActionResult, error) {
	driver := &actiontrail.Driver{Client: p.apiClient, Region: p.region, AccessKey: p.accessKey}
	switch action {
	case "dump":
		events, err := driver.DumpEvents(ctx, args)
		if err != nil {
			return schema.EventActionResult{}, err
		}
		return schema.EventActionResult{
			Action: "dump",
			Scope:  args,
			Events: events,
		}, nil
	case "whitelist":
		return driver.HandleEvents(ctx, args)
	default:
		return schema.EventActionResult{}, fmt.Errorf("invalid action: %s (expected: dump, whitelist)", action)
	}
}

// DBManagement implements schema.DBManager for JDCloud RDS. `useradd` /
// `userdel` create and revoke validation accounts under
// `/v1/regions/<region>/instances/<id>/accounts`.
func (p *Provider) DBManagement(ctx context.Context, action, instanceID string) (schema.DatabaseActionResult, error) {
	driver := &rds.Driver{Client: p.apiClient, Region: p.region}
	switch action {
	case "useradd":
		return driver.CreateAccount(ctx, instanceID)
	case "userdel":
		return driver.DeleteAccount(ctx, instanceID)
	default:
		return schema.DatabaseActionResult{}, fmt.Errorf("invalid action: %s (expected: useradd, userdel)", action)
	}
}

// ExecuteCloudVMCommand routes through JDCloud Cloud Assistant (assistant.jdcloud-api.com).
// Region must be a real VM region (cn-north-1 / cn-east-2 / ...); we resolve it
// from the host cache populated by `cloudlist` so `shell <instance-id>` works
// regardless of the session's current region setting.
func (p *Provider) ExecuteCloudVMCommand(ctx context.Context, instanceID, cmd string) (schema.CommandResult, error) {
	if osType, command, ok := vmexecspec.Parse(cmd); ok {
		if p.region == "" || p.region == "all" {
			return schema.CommandResult{}, fmt.Errorf("headless shell requires explicit region")
		}
		driver := &assistant.Driver{Client: p.apiClient, Region: p.region}
		output, err := driver.RunCommandContext(ctx, instanceID, osType, command)
		if err != nil {
			return schema.CommandResult{}, err
		}
		return schema.CommandResult{Output: output}, nil
	}

	if strings.HasPrefix(instanceID, "lavm-") {
		return schema.CommandResult{}, fmt.Errorf("JDCloud shell currently supports VM only")
	}
	host, ok := p.lookupHost(instanceID)
	if !ok {
		return schema.CommandResult{}, fmt.Errorf("unable to resolve instance metadata, run `cloudlist` first and retry")
	}
	command, err := base64.StdEncoding.DecodeString(cmd)
	if err != nil {
		return schema.CommandResult{}, err
	}
	driver := &assistant.Driver{Client: p.apiClient, Region: host.Region}
	output, err := driver.RunCommandContext(ctx, instanceID, host.OSType, string(command))
	if err != nil {
		return schema.CommandResult{}, err
	}
	return schema.CommandResult{Output: output}, nil
}

func (p *Provider) lookupHost(instanceID string) (schema.Host, bool) {
	for _, host := range vm.GetCacheHostList() {
		if host.ID == instanceID || host.HostName == instanceID {
			return host, true
		}
	}
	return schema.Host{}, false
}

func (p *Provider) newOSSDriver(region string) *oss.Driver {
	return &oss.Driver{
		Client:              p.apiClient,
		Credential:          p.credential,
		Region:              region,
		ObjectClientOptions: append([]awsapi.Option(nil), p.objectAPIOptions...),
	}
}

func (p *Provider) bucketInfos(ctx context.Context, driver *oss.Driver, bucketName string) (map[string]string, error) {
	infos := make(map[string]string)
	bucketName = strings.TrimSpace(bucketName)
	switch {
	case bucketName == "":
		return nil, fmt.Errorf("empty bucket name")
	case bucketName == "all":
		buckets, err := driver.ListBuckets(ctx)
		if err != nil {
			return nil, err
		}
		for _, bucket := range buckets {
			bucketRegion := strings.TrimSpace(bucket.Region)
			if bucketRegion == "" {
				bucketRegion, err = driver.ResolveBucketRegion(ctx, bucket.BucketName)
				if err != nil {
					return nil, err
				}
			}
			infos[bucket.BucketName] = bucketRegion
		}
		if len(infos) == 0 {
			return nil, fmt.Errorf("no buckets found")
		}
		return infos, nil
	case p.region != "" && p.region != "all":
		infos[bucketName] = p.region
		return infos, nil
	default:
		resolved, err := driver.ResolveBucketRegion(ctx, bucketName)
		if err != nil {
			return nil, fmt.Errorf("bucket %s region not found; set region explicitly or use `list all` first", bucketName)
		}
		infos[bucketName] = resolved
		return infos, nil
	}
}
