package huawei

import (
	"context"
	"fmt"
	"net/http"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	huaweiauth "github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
	_bss "github.com/404tk/cloudtoolkit/pkg/providers/huawei/bss"
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
	cred     huaweiauth.Credential
	regions  []string
	domainID string
}

var defaultRegion = "cn-north-4"

// New creates a new provider client for huawei API
func New(options schema.Options) (*Provider, error) {
	cred, err := huaweiauth.FromOptions(options)
	if err != nil {
		return nil, err
	}
	controlPlaneCred := cred
	domainID := ""
	provider := &Provider{cred: cred}
	if controlPlaneCred.Region == "all" {
		controlPlaneCred.Region = defaultRegion
	}

	payload, _ := options.GetMetadata(utils.Payload)
	if payload == "cloudlist" {
		probe := &_iam.Driver{Cred: controlPlaneCred}
		if err := credverify.ForCloudlist(options, provider, false, func(ctx context.Context) (credverify.Result, error) {
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
		client := api.NewClient(controlPlaneCred)
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

// Resources returns the provider for a resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	collector := schema.NewResourceCollector(p.Name()).
		Register("balance", func(ctx context.Context, _ *schema.Resources) {
			d := &_bss.Driver{Cred: p.cred}
			d.QueryAccountBalance(ctx)
		}).
		Register("host", func(ctx context.Context, list *schema.Resources) {
			ecsprovider := &ecs.Driver{Cred: p.iamCredential(), Regions: p.serviceRegions("ecs"), DomainID: p.domainID}
			hosts, err := ecsprovider.GetResource(ctx)
			schema.AppendAssets(list, hosts)
			list.AddError("host", err)
		}).
		Register("account", func(ctx context.Context, list *schema.Resources) {
			iamprovider := &_iam.Driver{Cred: p.iamCredential(), DomainID: p.domainID}
			users, err := iamprovider.ListUsers(ctx)
			schema.AppendAssets(list, users)
			list.AddError("account", err)
		}).
		Register("database", func(ctx context.Context, list *schema.Resources) {
			rdsprovider := &_rds.Driver{Cred: p.iamCredential(), Regions: p.serviceRegions("rds"), DomainID: p.domainID}
			databases, err := rdsprovider.GetDatabases(ctx)
			schema.AppendAssets(list, databases)
			list.AddError("database", err)
		}).
		Register("bucket", func(ctx context.Context, list *schema.Resources) {
			obsprovider := &_obs.Driver{Cred: p.iamCredential(), Regions: p.regions}
			storages, err := obsprovider.GetBuckets(ctx)
			schema.AppendAssets(list, storages)
			list.AddError("bucket", err)
		})

	return collector.Collect(ctx, env.From(ctx).Cloudlist)
}

func (p *Provider) UserManagement(action, username, password string) (schema.IAMResult, error) {
	r := &_iam.Driver{
		Cred: p.iamCredential(), Username: username, Password: password, DomainID: p.domainID}
	switch action {
	case "add":
		return r.AddUser()
	case "del":
		return r.DelUser()
	default:
		return schema.IAMResult{}, fmt.Errorf("invalid action: %s (expected: add, del)", action)
	}
}

func (p *Provider) BucketDump(ctx context.Context, action, bucketName string) ([]schema.BucketResult, error) {
	obsprovider := &_obs.Driver{Cred: p.iamCredential(), Regions: p.regions}

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

func (p *Provider) iamCredential() huaweiauth.Credential {
	cred := p.cred
	if cred.Region == "all" {
		cred.Region = defaultRegion
	}
	return cred
}
