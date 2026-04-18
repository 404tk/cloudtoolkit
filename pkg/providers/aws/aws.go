package aws

import (
	"context"
	"fmt"

	_api "github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	_auth "github.com/404tk/cloudtoolkit/pkg/providers/aws/auth"
	_ec2 "github.com/404tk/cloudtoolkit/pkg/providers/aws/ec2"
	_iam "github.com/404tk/cloudtoolkit/pkg/providers/aws/iam"
	_s3 "github.com/404tk/cloudtoolkit/pkg/providers/aws/s3"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/cloudtoolkit/utils/logger"
	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
)

// Provider is a data provider for aws API
type Provider struct {
	region    string
	cfg       awsv2.Config
	apiClient *_api.Client
}

// New creates a new provider client for aws API
func New(options schema.Options) (*Provider, error) {
	credential, err := _auth.FromOptions(options)
	if err != nil {
		return nil, err
	}
	region, _ := options.GetMetadata(utils.Region)
	version, _ := options.GetMetadata(utils.Version)
	cfg, err := newConfig(
		credential.AccessKeyID,
		credential.SecretAccessKey,
		credential.SessionToken,
		region,
		version,
	)
	if err != nil {
		return nil, err
	}
	apiClient := _api.NewClient(credential)

	payload, _ := options.GetMetadata(utils.Payload)
	if payload == "cloudlist" {
		resp, err := apiClient.GetCallerIdentity(
			context.Background(),
			resolveBootstrapRegion(region, version),
		)
		if err != nil {
			return nil, err
		}
		accountArn := resp.Arn
		userName := currentUserNameFromARN(accountArn)
		logger.Warning(fmt.Sprintf("Current user: %s", userName))
		cache.Cfg.CredInsert(userName, options)
	}

	return &Provider{
		region:    region,
		cfg:       cfg,
		apiClient: apiClient,
	}, nil
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return "aws"
}

// Resources returns the provider for an resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	list := schema.NewResources()
	list.Provider = p.Name()
	for _, product := range utils.Cloudlist {
		switch product {
		case "host":
			ec2provider := &_ec2.Driver{
				Client:        p.apiClient,
				Region:        p.region,
				DefaultRegion: p.cfg.Region,
			}
			hosts, err := ec2provider.GetResource(ctx)
			schema.AppendAssets(&list, hosts)
			list.AddError("host", err)
		case "account":
			iamprovider := &_iam.Driver{
				Client:        p.apiClient,
				Region:        p.region,
				DefaultRegion: p.cfg.Region,
			}
			users, err := iamprovider.ListUsers(ctx)
			schema.AppendAssets(&list, users)
			list.AddError("account", err)
		case "bucket":
			s3provider := &_s3.Driver{Client: p.apiClient, DefaultRegion: p.cfg.Region}
			storages, err := s3provider.GetBuckets(ctx)
			schema.AppendAssets(&list, storages)
			list.AddError("bucket", err)
		default:
		}
	}

	return list, list.Err()
}

func (p *Provider) UserManagement(action, username, password string) {
	ramprovider := &_iam.Driver{
		Client:        p.apiClient,
		Region:        p.region,
		DefaultRegion: p.cfg.Region,
		Username:      username,
		Password:      password,
	}
	switch action {
	case "add":
		ramprovider.AddUser()
	case "del":
		ramprovider.DelUser()
	default:
		logger.Error("Please set metadata like \"add username password\" or \"del username\"")
	}
}

func (p *Provider) BucketDump(ctx context.Context, action, bucketName string) {
	s3provider := &_s3.Driver{Client: p.apiClient, DefaultRegion: p.cfg.Region}
	switch action {
	case "list":
		var infos = make(map[string]string)
		if bucketName == "all" {
			buckets, err := s3provider.GetBuckets(context.Background())
			if err != nil {
				logger.Error("List buckets failed:", err)
				return
			}
			for _, b := range buckets {
				infos[b.BucketName] = b.Region
			}
		} else {
			infos[bucketName] = p.cfg.Region
		}
		s3provider.ListObjects(ctx, infos)
	case "total":
		var infos = make(map[string]string)
		if bucketName == "all" {
			buckets, err := s3provider.GetBuckets(context.Background())
			if err != nil {
				logger.Error("List buckets failed:", err)
				return
			}
			for _, b := range buckets {
				infos[b.BucketName] = b.Region
			}
		} else {
			infos[bucketName] = p.cfg.Region
		}
		s3provider.TotalObjects(ctx, infos)
	default:
		logger.Error("`list all` or `total all`.")
	}
}
