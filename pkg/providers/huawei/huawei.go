package huawei

import (
	"context"
	"net/http"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	huaweiauth "github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
	_bss "github.com/404tk/cloudtoolkit/pkg/providers/huawei/bss"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/ecs"
	_iam "github.com/404tk/cloudtoolkit/pkg/providers/huawei/iam"
	_obs "github.com/404tk/cloudtoolkit/pkg/providers/huawei/obs"
	_rds "github.com/404tk/cloudtoolkit/pkg/providers/huawei/rds"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
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
	if controlPlaneCred.Region == "all" {
		controlPlaneCred.Region = defaultRegion
	}

	payload, _ := options.GetMetadata(utils.Payload)
	if payload == "cloudlist" {
		probe := &_iam.Driver{Cred: controlPlaneCred}
		userName, err := probe.GetUserName(context.Background())
		if err != nil {
			return nil, err
		}
		msg := "Current user: " + userName
		cache.Cfg.CredInsert(userName, options)
		logger.Warning(msg)
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

	return &Provider{
		cred:     cred,
		regions:  regions,
		domainID: domainID,
	}, nil
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return "huawei"
}

// Resources returns the provider for a resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	list := schema.NewResources()
	list.Provider = p.Name()
	for _, product := range utils.Cloudlist {
		switch product {
		case "balance":
			d := &_bss.Driver{Cred: p.cred}
			d.QueryAccountBalance(ctx)
		case "host":
			ecsprovider := &ecs.Driver{Cred: p.iamCredential(), Regions: p.serviceRegions("ecs"), DomainID: p.domainID}
			hosts, err := ecsprovider.GetResource(ctx)
			schema.AppendAssets(&list, hosts)
			list.AddError("host", err)
		case "account":
			iamprovider := &_iam.Driver{Cred: p.iamCredential(), DomainID: p.domainID}
			users, err := iamprovider.ListUsers(ctx)
			schema.AppendAssets(&list, users)
			list.AddError("account", err)
		case "database":
			rdsprovider := &_rds.Driver{Cred: p.iamCredential(), Regions: p.serviceRegions("rds"), DomainID: p.domainID}
			databases, err := rdsprovider.GetDatabases(ctx)
			schema.AppendAssets(&list, databases)
			list.AddError("database", err)
		case "bucket":
			obsprovider := &_obs.Driver{Cred: p.iamCredential(), Regions: p.regions}
			storages, err := obsprovider.GetBuckets(ctx)
			schema.AppendAssets(&list, storages)
			list.AddError("bucket", err)
		default:
		}
	}

	return list, list.Err()
}

func (p *Provider) UserManagement(action, username, password string) {
	r := &_iam.Driver{
		Cred: p.iamCredential(), Username: username, Password: password, DomainID: p.domainID}
	switch action {
	case "add":
		r.AddUser()
	case "del":
		r.DelUser()
	default:
		logger.Error("Please set metadata like \"add username password\" or \"del username\"")
	}
}

func (p *Provider) BucketDump(ctx context.Context, action, bucketName string) {
	obsprovider := &_obs.Driver{Cred: p.iamCredential(), Regions: p.regions}
	switch action {
	case "list":
		infos := make(map[string]string)
		if bucketName == "all" {
			buckets, err := obsprovider.GetBuckets(context.Background())
			if err != nil {
				logger.Error("List buckets failed:", err)
				return
			}
			for _, bucket := range buckets {
				infos[bucket.BucketName] = bucket.Region
			}
		} else {
			infos[bucketName] = p.iamCredential().Region
		}
		obsprovider.ListObjects(ctx, infos)
	case "total":
		infos := make(map[string]string)
		if bucketName == "all" {
			buckets, err := obsprovider.GetBuckets(context.Background())
			if err != nil {
				logger.Error("List buckets failed:", err)
				return
			}
			for _, bucket := range buckets {
				infos[bucket.BucketName] = bucket.Region
			}
		} else {
			infos[bucketName] = p.iamCredential().Region
		}
		obsprovider.TotalObjects(ctx, infos)
	default:
		logger.Error("`list all` or `total all`.")
	}
}

func (p *Provider) iamCredential() huaweiauth.Credential {
	cred := p.cred
	if cred.Region == "all" {
		cred.Region = defaultRegion
	}
	return cred
}
