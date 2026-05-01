package huawei

import (
	"context"
	"fmt"
	"net/http"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	huaweiauth "github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
	_bss "github.com/404tk/cloudtoolkit/pkg/providers/huawei/bss"
	_cts "github.com/404tk/cloudtoolkit/pkg/providers/huawei/cts"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/ecs"
	_iam "github.com/404tk/cloudtoolkit/pkg/providers/huawei/iam"
	_obs "github.com/404tk/cloudtoolkit/pkg/providers/huawei/obs"
	_rds "github.com/404tk/cloudtoolkit/pkg/providers/huawei/rds"
	"github.com/404tk/cloudtoolkit/pkg/providers/internal/credverify"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

// Provider is a data provider for huawei API
type Provider struct {
	cred       huaweiauth.Credential
	regions    []string
	domainID   string
	apiOptions []api.Option
	obsOptions []_obs.Option
	skipCache  bool
}

var defaultRegion = "cn-north-4"

// ClientConfig allows callers (e.g. demo replay) to inject custom api.Option
// and obs.Option values and skip credential cache writes for ephemeral
// credentials.
type ClientConfig struct {
	APIOptions          []api.Option
	OBSOptions          []_obs.Option
	SkipCredentialCache bool
}

// New creates a new provider client for huawei API
func New(options schema.Options) (*Provider, error) {
	return NewWithConfig(options, ClientConfig{})
}

// NewWithConfig creates a new provider client for huawei API with injected
// transport options. Real callers use New; replay/test callers feed in a
// mock HTTP client through cfg.APIOptions / cfg.OBSOptions.
func NewWithConfig(options schema.Options, cfg ClientConfig) (*Provider, error) {
	cred, err := huaweiauth.FromOptions(options)
	if err != nil {
		return nil, err
	}
	controlPlaneCred := cred
	domainID := ""
	provider := &Provider{
		cred:       cred,
		apiOptions: append([]api.Option(nil), cfg.APIOptions...),
		obsOptions: append([]_obs.Option(nil), cfg.OBSOptions...),
		skipCache:  cfg.SkipCredentialCache,
	}
	if controlPlaneCred.Region == "all" {
		controlPlaneCred.Region = defaultRegion
	}

	payload, _ := options.GetMetadata(utils.Payload)
	if payload == "cloudlist" {
		probe := &_iam.Driver{Cred: controlPlaneCred, Client: provider.newAPIClient(controlPlaneCred)}
		if err := credverify.ForCloudlist(options, provider, cfg.SkipCredentialCache, func(ctx context.Context) (credverify.Result, error) {
			userName, err := probe.GetUserName(ctx)
			if err != nil {
				return credverify.Result{}, err
			}
			return credverify.Result{
				Summary:     "Current user: " + userName,
				SessionUser: userName,
			}, nil
		}); err != nil {
			return nil, err
		}
		domainID = probe.DomainID
	}

	var regions []string
	if cred.Region == "all" && payload == "cloudlist" {
		client := provider.newAPIClient(controlPlaneCred)
		var resp api.ListRegionsResponse
		err := client.DoJSON(context.Background(), api.Request{
			Service:    "iam",
			Region:     defaultRegion,
			Intl:       cred.Intl,
			Method:     http.MethodGet,
			Path:       "/v3/regions",
			Idempotent: true,
		}, &resp)
		if err != nil {
			logger.Error("List regions failed.")
			return nil, err
		}
		for _, r := range resp.Regions {
			regions = append(regions, r.ID)
		}
	} else if cred.Region == "all" && payload != "cloudlist" {
		regions = append(regions, defaultRegion)
	} else {
		regions = append(regions, cred.Region)
	}

	provider.regions = regions
	provider.domainID = domainID
	return provider, nil
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return "huawei"
}

func (p *Provider) newAPIClient(cred huaweiauth.Credential) *api.Client {
	return api.NewClient(cred, p.apiOptions...)
}

func (p *Provider) newOBSClient(cred huaweiauth.Credential) *_obs.Client {
	return _obs.NewClient(cred, p.obsOptions...)
}

// Resources returns the provider for a resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	collector := schema.NewResourceCollector(p.Name()).
		Register("balance", func(ctx context.Context, _ *schema.Resources) {
			d := &_bss.Driver{Cred: p.cred, Client: p.newAPIClient(p.cred)}
			d.QueryAccountBalance(ctx)
		}).
		Register("host", func(ctx context.Context, list *schema.Resources) {
			cred := p.iamCredential()
			ecsprovider := &ecs.Driver{Cred: cred, Regions: p.serviceRegions("ecs"), DomainID: p.domainID, Client: p.newAPIClient(cred)}
			hosts, err := ecsprovider.GetResource(ctx)
			schema.AppendAssets(list, hosts)
			list.AddError("host", err)
		}).
		Register("account", func(ctx context.Context, list *schema.Resources) {
			cred := p.iamCredential()
			iamprovider := &_iam.Driver{Cred: cred, DomainID: p.domainID, Client: p.newAPIClient(cred)}
			users, err := iamprovider.ListUsers(ctx)
			schema.AppendAssets(list, users)
			list.AddError("account", err)
		}).
		Register("database", func(ctx context.Context, list *schema.Resources) {
			cred := p.iamCredential()
			rdsprovider := &_rds.Driver{Cred: cred, Regions: p.serviceRegions("rds"), DomainID: p.domainID, Client: p.newAPIClient(cred)}
			databases, err := rdsprovider.GetDatabases(ctx)
			schema.AppendAssets(list, databases)
			list.AddError("database", err)
		}).
		Register("bucket", func(ctx context.Context, list *schema.Resources) {
			cred := p.iamCredential()
			obsprovider := &_obs.Driver{Cred: cred, Regions: p.regions, Client: p.newOBSClient(cred)}
			storages, err := obsprovider.GetBuckets(ctx)
			schema.AppendAssets(list, storages)
			list.AddError("bucket", err)
		})

	return collector.Collect(ctx, env.From(ctx).Cloudlist)
}

func (p *Provider) UserManagement(action, username, password string) (schema.IAMResult, error) {
	cred := p.iamCredential()
	r := &_iam.Driver{
		Cred: cred, Username: username, Password: password, DomainID: p.domainID, Client: p.newAPIClient(cred)}
	switch action {
	case "add":
		return r.AddUser()
	case "del":
		return r.DelUser()
	default:
		return schema.IAMResult{}, fmt.Errorf("invalid action: %s (expected: add, del)", action)
	}
}

// RoleBinding implements schema.RoleBindingManager for huawei IAM. Huawei has
// no direct user-policy attachment; policies live on keystone groups and users
// gain permissions by joining groups. The capability therefore models group
// membership: `principal` is the user name, `role` is the group name, `scope`
// is reserved (membership is domain-scoped via the X-Domain-Id header).
func (p *Provider) RoleBinding(ctx context.Context, action, principal, role, scope string) (schema.RoleBindingResult, error) {
	cred := p.iamCredential()
	driver := &_iam.Driver{Cred: cred, DomainID: p.domainID, Client: p.newAPIClient(cred)}
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
		result.Message = fmt.Sprintf("%d groups for user %s", len(bindings), principal)
		return result, nil
	case "add":
		if err := driver.AttachGroup(ctx, principal, role); err != nil {
			return result, err
		}
		result.Message = fmt.Sprintf("added user %s to group %s", principal, role)
		return result, nil
	case "del":
		if err := driver.DetachGroup(ctx, principal, role); err != nil {
			return result, err
		}
		result.Message = fmt.Sprintf("removed user %s from group %s", principal, role)
		return result, nil
	}
	return result, fmt.Errorf("huawei: unsupported role-binding action %q", action)
}

func (p *Provider) BucketDump(ctx context.Context, action, bucketName string) ([]schema.BucketResult, error) {
	cred := p.iamCredential()
	obsprovider := &_obs.Driver{Cred: cred, Regions: p.regions, Client: p.newOBSClient(cred)}

	infos := make(map[string]string)
	if bucketName == "all" {
		buckets, err := obsprovider.GetBuckets(context.Background())
		if err != nil {
			return nil, fmt.Errorf("list buckets: %w", err)
		}
		for _, bucket := range buckets {
			infos[bucket.BucketName] = bucket.Region
		}
	} else {
		// For a specific bucket, we need to find its region first
		buckets, err := obsprovider.GetBuckets(context.Background())
		if err != nil {
			return nil, fmt.Errorf("list buckets: %w", err)
		}
		found := false
		for _, bucket := range buckets {
			if bucket.BucketName == bucketName {
				infos[bucketName] = bucket.Region
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("bucket %s not found", bucketName)
		}
	}

	switch action {
	case "list":
		return obsprovider.ListObjects(ctx, infos)
	case "total":
		return obsprovider.TotalObjects(ctx, infos)
	default:
		return nil, fmt.Errorf("invalid action: %s (expected: list, total)", action)
	}
}

// BucketACL implements schema.BucketACLManager for huawei OBS. `level`
// accepts canned OBS ACL values (private / public-read / public-read-write
// + the *-delivered variants) or friendly aliases resolved by
// obs.NormalizeOBSACL.
func (p *Provider) BucketACL(ctx context.Context, action, container, level string) (schema.BucketACLResult, error) {
	cred := p.iamCredential()
	driver := &_obs.Driver{Cred: cred, Regions: p.regions, Client: p.newOBSClient(cred)}
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
		result.Level = _obs.OBSACLPrivate
		result.Message = fmt.Sprintf("bucket %s reverted to private", container)
		return result, nil
	}
	return result, fmt.Errorf("huawei: unsupported bucket-acl action %q", action)
}

// EventDump implements schema.EventReader for Huawei CTS. The `dump` action
// lists recent management traces; `whitelist` returns a clear unsupported
// error because CTS is a read-only audit service.
func (p *Provider) EventDump(ctx context.Context, action, args string) (schema.EventActionResult, error) {
	cred := p.iamCredential()
	driver := &_cts.Driver{
		Cred:     cred,
		Regions:  p.regions,
		DomainID: p.domainID,
		Client:   p.newAPIClient(cred),
	}
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

func (p *Provider) iamCredential() huaweiauth.Credential {
	cred := p.cred
	if cred.Region == "all" {
		cred.Region = defaultRegion
	}
	return cred
}
