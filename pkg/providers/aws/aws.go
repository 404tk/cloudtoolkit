package aws

import (
	"context"
	"fmt"

	_api "github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	_auth "github.com/404tk/cloudtoolkit/pkg/providers/aws/auth"
	_ec2 "github.com/404tk/cloudtoolkit/pkg/providers/aws/ec2"
	_iam "github.com/404tk/cloudtoolkit/pkg/providers/aws/iam"
	_s3 "github.com/404tk/cloudtoolkit/pkg/providers/aws/s3"
	"github.com/404tk/cloudtoolkit/pkg/providers/internal/credverify"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
)

// Provider is a data provider for aws API
type Provider struct {
	region        string
	defaultRegion string
	apiClient     *_api.Client
}

// New creates a new provider client for aws API
func New(options schema.Options) (*Provider, error) {
	credential, err := _auth.FromOptions(options)
	if err != nil {
		return nil, err
	}
	if err := credential.Validate(); err != nil {
		return nil, err
	}
	region, _ := options.GetMetadata(utils.Region)
	version, _ := options.GetMetadata(utils.Version)
	defaultRegion := resolveBootstrapRegion(region, version)
	apiClient := _api.NewClient(credential)
	provider := &Provider{
		region:        region,
		defaultRegion: defaultRegion,
		apiClient:     apiClient,
	}

	if err := credverify.ForCloudlist(options, provider, false, func(ctx context.Context) (credverify.Result, error) {
		resp, err := apiClient.GetCallerIdentity(ctx, defaultRegion)
		if err != nil {
			return credverify.Result{}, err
		}
		userName := currentUserNameFromARN(resp.Arn)
		return credverify.Result{
			Summary:     fmt.Sprintf("Current user: %s", userName),
			SessionUser: userName,
		}, nil
	}); err != nil {
		return nil, err
	}

	return provider, nil
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return "aws"
}

// Resources returns the provider for an resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	collector := schema.NewResourceCollector(p.Name()).
		Register("host", func(ctx context.Context, list *schema.Resources) {
			ec2provider := &_ec2.Driver{
				Client:        p.apiClient,
				Region:        p.region,
				DefaultRegion: p.defaultRegion,
			}
			hosts, err := ec2provider.GetResource(ctx)
			schema.AppendAssets(list, hosts)
			list.AddError("host", err)
			list.AddError("host", ec2provider.PartialError())
		}).
		Register("account", func(ctx context.Context, list *schema.Resources) {
			iamprovider := &_iam.Driver{
				Client:        p.apiClient,
				Region:        p.region,
				DefaultRegion: p.defaultRegion,
			}
			users, err := iamprovider.ListUsers(ctx)
			schema.AppendAssets(list, users)
			list.AddError("account", err)
		}).
		Register("bucket", func(ctx context.Context, list *schema.Resources) {
			s3provider := &_s3.Driver{Client: p.apiClient, DefaultRegion: p.defaultRegion}
			storages, err := s3provider.GetBuckets(ctx)
			schema.AppendAssets(list, storages)
			list.AddError("bucket", err)
		})

	return collector.Collect(ctx, env.From(ctx).Cloudlist)
}

func (p *Provider) UserManagement(action, username, password string) (schema.IAMResult, error) {
	ramprovider := &_iam.Driver{
		Client:        p.apiClient,
		Region:        p.region,
		DefaultRegion: p.defaultRegion,
		Username:      username,
		Password:      password,
	}
	switch action {
	case "add":
		return ramprovider.AddUser()
	case "del":
		return ramprovider.DelUser()
	default:
		return schema.IAMResult{}, fmt.Errorf("invalid action: %s (expected: add, del)", action)
	}
}

func (p *Provider) BucketDump(ctx context.Context, action, bucketName string) ([]schema.BucketResult, error) {
	s3provider := &_s3.Driver{Client: p.apiClient, DefaultRegion: p.defaultRegion}
	switch action {
	case "list":
		var infos = make(map[string]string)
		if bucketName == "all" {
			buckets, err := s3provider.GetBuckets(context.Background())
			if err != nil {
				return nil, fmt.Errorf("list buckets: %w", err)
			}
			for _, b := range buckets {
				infos[b.BucketName] = b.Region
			}
		} else {
			infos[bucketName] = p.defaultRegion
		}
		return s3provider.ListObjects(ctx, infos)
	case "total":
		var infos = make(map[string]string)
		if bucketName == "all" {
			buckets, err := s3provider.GetBuckets(context.Background())
			if err != nil {
				return nil, fmt.Errorf("list buckets: %w", err)
			}
			for _, b := range buckets {
				infos[b.BucketName] = b.Region
			}
		} else {
			infos[bucketName] = p.defaultRegion
		}
		return s3provider.TotalObjects(ctx, infos)
	default:
		return nil, fmt.Errorf("invalid action: %s (expected: list, total)", action)
	}
}
