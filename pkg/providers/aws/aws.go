package aws

import (
	"context"

	_ec2 "github.com/404tk/cloudtoolkit/pkg/providers/aws/ec2"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// Provider is a data provider for aws API
type Provider struct {
	vendor    string
	regions   []string
	EC2Client *ec2.EC2
	session   *session.Session
}

// New creates a new provider client for aws API
func New(options schema.OptionBlock) (*Provider, error) {
	accessKey, ok := options.GetMetadata(utils.AccessKey)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: utils.AccessKey}
	}
	secretKey, ok := options.GetMetadata(utils.SecretKey)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: utils.SecretKey}
	}
	conf := aws.NewConfig()
	token, _ := options.GetMetadata(utils.SessionToken)
	region, _ := options.GetMetadata(utils.Region)
	if region == "" {
		if v, _ := options.GetMetadata("version"); v == "China" {
			conf.WithRegion("cn-northwest-1")
		} else {
			conf.WithRegion("us-east-1")
		}
	} else {
		conf.WithRegion(region)
	}
	conf.WithCredentials(credentials.NewStaticCredentials(accessKey, secretKey, token))

	session, err := session.NewSession(conf)
	if err != nil {
		return nil, err
	}

	ec2Client := ec2.New(session)
	var regions []string
	if region == "" {
		resp, err := ec2Client.DescribeRegions(&ec2.DescribeRegionsInput{})
		if err != nil {
			return nil, err
		}
		for _, region := range resp.Regions {
			regions = append(regions, aws.StringValue(region.RegionName))
		}
	} else {
		regions = append(regions, region)
	}

	return &Provider{
		vendor:    "aws",
		regions:   regions,
		EC2Client: ec2Client,
		session:   session,
	}, nil
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return p.vendor
}

// Resources returns the provider for an resource deployment source.
func (p *Provider) Resources(ctx context.Context) (*schema.Resources, error) {
	list := schema.NewResources()
	list.Provider = p.vendor
	ec2provider := &_ec2.InstanceProvider{
		Ec2Client: p.EC2Client, Session: p.session, Regions: p.regions}
	list.Hosts, _ = ec2provider.GetResource(ctx)

	return list, nil
}
