package aws

import (
	"context"
	"log"
	"strings"

	_ec2 "github.com/404tk/cloudtoolkit/pkg/providers/aws/ec2"
	_iam "github.com/404tk/cloudtoolkit/pkg/providers/aws/iam"
	_s3 "github.com/404tk/cloudtoolkit/pkg/providers/aws/s3"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

// Provider is a data provider for aws API
type Provider struct {
	vendor  string
	region  string
	session *session.Session
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
	token, _ := options.GetMetadata(utils.SecurityToken)
	region, _ := options.GetMetadata(utils.Region)
	if region == "all" {
		if v, _ := options.GetMetadata(utils.Version); v == "China" {
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

	// Get current username
	stsclient := sts.New(session)
	resp, err := stsclient.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}
	accountArn := *resp.Arn
	var userName string
	if len(accountArn) >= 4 && accountArn[len(accountArn)-4:] == "root" {
		userName = "root"
	} else {
		if u := strings.Split(accountArn, "/"); len(u) > 1 {
			userName = u[1]
		}
	}
	log.Printf("[+] Current user: %s\n", userName)
	cache.Cfg.CredInsert(userName, options)

	return &Provider{
		vendor:  "aws",
		region:  region,
		session: session,
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
	var err error
	ec2provider := &_ec2.InstanceProvider{Session: p.session, Region: p.region}
	list.Hosts, err = ec2provider.GetResource(ctx)

	s3provider := &_s3.S3Provider{Session: p.session}
	list.Storages, err = s3provider.GetBuckets(ctx)

	iamprovider := &_iam.IAMProvider{Session: p.session}
	list.Users, err = iamprovider.GetIAMUser(ctx)

	return list, err
}

func (p *Provider) UserManagement(action, uname, pwd string) {}
