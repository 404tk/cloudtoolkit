package volcengine

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/billing"
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/ecs"
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/iam"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/volcengine/volcengine-go-sdk/service/iam20210801"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/credentials"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
)

type Provider struct {
	conf   *volcengine.Config
	region string
}

// New creates a new provider client for alibaba API
func New(options schema.Options) (*Provider, error) {
	accessKey, ok := options.GetMetadata(utils.AccessKey)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: utils.AccessKey}
	}
	secretKey, ok := options.GetMetadata(utils.SecretKey)
	if !ok {
		return nil, &schema.ErrNoSuchKey{Name: utils.SecretKey}
	}
	region, _ := options.GetMetadata(utils.Region)
	token, _ := options.GetMetadata(utils.SecurityToken)
	cred := credentials.NewStaticCredentials(accessKey, secretKey, token)
	config := volcengine.NewConfig().WithCredentials(cred)

	payload, _ := options.GetMetadata(utils.Payload)
	if payload == "cloudlist" {
		name, err := getProject(region, config)
		if err != nil {
			return nil, err
		}
		cache.Cfg.CredInsert(name, options)
	}

	return &Provider{
		conf:   config,
		region: region,
	}, nil
}

func getProject(r string, conf *volcengine.Config) (string, error) {
	if r == "all" {
		conf = conf.WithRegion("cn-beijing")
	} else {
		conf = conf.WithRegion(r)
	}
	sess, _ := session.NewSession(conf)
	svc := iam20210801.New(sess)
	out, err := svc.ListProjects(&iam20210801.ListProjectsInput{})
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s(%d)", *out.Projects[0].ProjectName, *out.Projects[0].AccountID)
	logger.Warning("Current project:", name)
	return name, nil
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return "volcengine"
}

// Resources returns the provider for a resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	list := schema.NewResources()
	list.Provider = p.Name()
	var err error
	for _, product := range utils.Cloudlist {
		switch product {
		case "balance":
			billing.QueryAccountBalance(p.conf)
		case "host":
			d := &ecs.Driver{Conf: p.conf, Region: p.region}
			list.Hosts, err = d.GetResource(ctx)
		case "domain":
		case "account":
			d := &iam.Driver{Conf: p.conf}
			list.Users, err = d.ListUsers(ctx)
		case "database":
		case "bucket":
		case "sms":
		case "log":
		default:
		}
	}

	return list, err
}

func (p *Provider) UserManagement(action, args_1, args_2 string) {}

func (p *Provider) BucketDump(ctx context.Context, action, bucketname string) {}

func (p *Provider) EventDump(action, args string) {}

func (p *Provider) ExecuteCloudVMCommand(instanceId, cmd string) {}

func (p *Provider) DBManagement(action, args string) {}
