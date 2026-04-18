package aws

import (
	"context"
	"fmt"

	_ec2 "github.com/404tk/cloudtoolkit/pkg/providers/aws/ec2"
	_iam "github.com/404tk/cloudtoolkit/pkg/providers/aws/iam"
	_s3 "github.com/404tk/cloudtoolkit/pkg/providers/aws/s3"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/cloudtoolkit/utils/logger"
	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	sts "github.com/aws/aws-sdk-go-v2/service/sts"
)

// Provider is a data provider for aws API
type Provider struct {
	region string
	cfg    awsv2.Config
}

// New creates a new provider client for aws API
func New(options schema.Options) (*Provider, error) {
	accessKey, ok := options.GetMetadata(utils.AccessKey)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: utils.AccessKey}
	}
	secretKey, ok := options.GetMetadata(utils.SecretKey)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: utils.SecretKey}
	}
	token, _ := options.GetMetadata(utils.SecurityToken)
	region, _ := options.GetMetadata(utils.Region)
	version, _ := options.GetMetadata(utils.Version)
	cfg, err := newConfig(context.Background(), accessKey, secretKey, token, region, version)
	if err != nil {
		return nil, err
	}

	payload, _ := options.GetMetadata(utils.Payload)
	if payload == "cloudlist" {
		stsclient := sts.NewFromConfig(cfg)
		resp, err := stsclient.GetCallerIdentity(context.Background(), &sts.GetCallerIdentityInput{})
		if err != nil {
			return nil, err
		}
		accountArn := awsv2.ToString(resp.Arn)
		userName := currentUserNameFromARN(accountArn)
		logger.Warning(fmt.Sprintf("Current user: %s", userName))
		cache.Cfg.CredInsert(userName, options)
	}

	return &Provider{
		region: region,
		cfg:    cfg,
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
			ec2provider := &_ec2.Driver{Config: p.cfg, Region: p.region}
			hosts, err := ec2provider.GetResource(ctx)
			schema.AppendAssets(&list, hosts)
			list.AddError("host", err)
		case "account":
			iamprovider := &_iam.Driver{Config: p.cfg}
			users, err := iamprovider.ListUsers(ctx)
			schema.AppendAssets(&list, users)
			list.AddError("account", err)
		case "bucket":
			s3provider := &_s3.Driver{Config: p.cfg}
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
		Config: p.cfg, Username: username, Password: password}
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
	s3provider := &_s3.Driver{Config: p.cfg}
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
