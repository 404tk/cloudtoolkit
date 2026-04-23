package jdcloud

import (
	"context"
	"encoding/base64"
	"fmt"

	_api "github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/assistant"
	_auth "github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/iam"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/oss"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/vm"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Provider struct {
	region    string
	accessKey string
	apiClient *_api.Client
}

// New creates a new provider client for JDCloud API.
func New(options schema.Options) (*Provider, error) {
	credential, err := _auth.FromOptions(options)
	if err != nil {
		return nil, err
	}
	region, _ := options.GetMetadata(utils.Region)
	apiClient := _api.NewClient(credential)
	payload, _ := options.GetMetadata(utils.Payload)

	if payload == "cloudlist" {
		d := &iam.Driver{Client: apiClient, AccessKey: credential.AccessKey}
		pin, ok := d.Validator()
		if !ok {
			return nil, fmt.Errorf("invalid accesskey")
		}
		if pin != "" {
			logger.Warning(fmt.Sprintf("Current user: %s", pin))
		}
		sessionUser := pin
		if sessionUser == "" {
			sessionUser = "default"
		}
		cache.Cfg.CredInsert(sessionUser, options)
	}

	return &Provider{
		region:    region,
		accessKey: credential.AccessKey,
		apiClient: apiClient,
	}, nil
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return "jdcloud"
}

// Resources returns the provider for a resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	list := schema.NewResources()
	list.Provider = p.Name()
	for _, product := range utils.Cloudlist {
		switch product {
		case "balance":
		case "host":
			d := &vm.Driver{Client: p.apiClient, Region: p.region}
			hosts, err := d.GetResource(ctx)
			schema.AppendAssets(&list, hosts)
			list.AddError("host", err)
		case "domain":
		case "account":
			d := &iam.Driver{Client: p.apiClient}
			users, err := d.ListUsers(ctx)
			schema.AppendAssets(&list, users)
			list.AddError("account", err)
		case "database":
		case "bucket":
			d := &oss.Driver{Client: p.apiClient}
			storages, err := d.ListBuckets(ctx)
			schema.AppendAssets(&list, storages)
			list.AddError("bucket", err)
		case "sms":
		case "log":
		default:
		}
	}

	return list, list.Err()
}

// UserManagement powers the iam-user-check payload. JDCloud's CreateSubUser is
// atomic (name + password + consoleLogin in one call), so we only need an
// AttachSubUserPolicy follow-up to grant administrator privilege.
func (p *Provider) UserManagement(action, username, password string) {
	driver := &iam.Driver{
		Client:    p.apiClient,
		AccessKey: p.accessKey,
		UserName:  username,
		Password:  password,
	}
	switch action {
	case "add":
		driver.AddUser()
	case "del":
		driver.DelUser()
	default:
		logger.Error("Please set metadata like \"add username password\" or \"del username\"")
	}
}

// ExecuteCloudVMCommand routes through JDCloud Cloud Assistant (assistant.jdcloud-api.com).
// Region must be a real VM region (cn-north-1 / cn-east-2 / ...); we resolve it
// from the host cache populated by `cloudlist` so `shell <instance-id>` works
// regardless of the session's current region setting.
func (p *Provider) ExecuteCloudVMCommand(instanceID, cmd string) {
	host, ok := p.lookupHost(instanceID)
	if !ok {
		logger.Error("Unable to resolve instance metadata, run `cloudlist` first and retry.")
		return
	}
	command, err := base64.StdEncoding.DecodeString(cmd)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	driver := &assistant.Driver{Client: p.apiClient, Region: host.Region}
	output := driver.RunCommand(instanceID, host.OSType, string(command))
	if output != "" {
		fmt.Println(output)
	}
}

func (p *Provider) lookupHost(instanceID string) (schema.Host, bool) {
	for _, host := range vm.GetCacheHostList() {
		if host.ID == instanceID || host.HostName == instanceID {
			return host, true
		}
	}
	return schema.Host{}, false
}
