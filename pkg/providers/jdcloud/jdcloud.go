package jdcloud

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/internal/credverify"
	_api "github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/asset"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/assistant"
	_auth "github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/iam"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/lavm"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/oss"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/vm"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/runtime/vmexecspec"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
)

type Provider struct {
	credential _auth.Credential
	region     string
	accessKey  string
	apiClient  *_api.Client
}

// New creates a new provider client for JDCloud API.
func New(options schema.Options) (*Provider, error) {
	credential, err := _auth.FromOptions(options)
	if err != nil {
		return nil, err
	}
	region, _ := options.GetMetadata(utils.Region)
	region = strings.TrimSpace(region)
	if strings.EqualFold(region, "all") {
		region = "all"
	}
	apiClient := _api.NewClient(credential)
	provider := &Provider{
		credential: credential,
		region:     region,
		accessKey:  credential.AccessKey,
		apiClient:  apiClient,
	}
	if err := credverify.ForCloudlist(options, provider, false, func(context.Context) (credverify.Result, error) {
		d := &iam.Driver{Client: apiClient, AccessKey: credential.AccessKey}
		pin, ok := d.Validator()
		if !ok {
			return credverify.Result{}, fmt.Errorf("invalid accesskey")
		}
		sessionUser := pin
		if sessionUser == "" {
			sessionUser = "default"
		}
		summary := ""
		if pin != "" {
			summary = fmt.Sprintf("Current user: %s", pin)
		}
		return credverify.Result{
			Summary:     summary,
			SessionUser: sessionUser,
		}, nil
	}); err != nil {
		return nil, err
	}

	return provider, nil
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return "jdcloud"
}

// Resources returns the provider for a resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	collector := schema.NewResourceCollector(p.Name()).
		Register("balance", func(ctx context.Context, _ *schema.Resources) {
			(&asset.Driver{Client: p.apiClient, Region: p.region}).QueryAccountBalance(ctx)
		}).
		Register("host", func(ctx context.Context, list *schema.Resources) {
			vmDriver := &vm.Driver{Client: p.apiClient, Region: p.region}
			vmHosts, vmErr := vmDriver.GetResource(ctx)
			schema.AppendAssets(list, vmHosts)
			list.AddError("host/vm", vmErr)

			lavmDriver := &lavm.Driver{Client: p.apiClient, Region: p.region}
			lavmHosts, lavmErr := lavmDriver.GetResource(ctx)
			schema.AppendAssets(list, lavmHosts)
			list.AddError("host/lavm", lavmErr)

			allHosts := append([]schema.Host{}, vmHosts...)
			allHosts = append(allHosts, lavmHosts...)
			if len(allHosts) > 0 || (vmErr == nil && lavmErr == nil) {
				vm.SetCacheHostList(allHosts)
			}
		}).
		Register("account", func(ctx context.Context, list *schema.Resources) {
			d := &iam.Driver{Client: p.apiClient}
			users, err := d.ListUsers(ctx)
			schema.AppendAssets(list, users)
			list.AddError("account", err)
		}).
		Register("bucket", func(ctx context.Context, list *schema.Resources) {
			d := p.newOSSDriver(p.region)
			storages, err := d.ListBuckets(ctx)
			schema.AppendAssets(list, storages)
			list.AddError("bucket", err)
		})

	return collector.Collect(ctx, env.From(ctx).Cloudlist)
}

// UserManagement powers the iam-user-check payload. JDCloud's CreateSubUser is
// atomic (name + password + consoleLogin in one call), so we only need an
// AttachSubUserPolicy follow-up to grant administrator privilege.
func (p *Provider) UserManagement(action, username, password string) (schema.IAMResult, error) {
	driver := &iam.Driver{
		Client:    p.apiClient,
		AccessKey: p.accessKey,
		UserName:  username,
		Password:  password,
	}
	switch action {
	case "add":
		return driver.AddUser()
	case "del":
		return driver.DelUser()
	default:
		return schema.IAMResult{}, fmt.Errorf("invalid action: %s (expected: add, del)", action)
	}
}

func (p *Provider) BucketDump(ctx context.Context, action, bucketName string) ([]schema.BucketResult, error) {
	driver := p.newOSSDriver(p.region)
	infos, err := p.bucketInfos(context.Background(), driver, bucketName)
	if err != nil {
		return nil, fmt.Errorf("list buckets: %w", err)
	}

	switch action {
	case "list":
		return driver.ListObjects(ctx, infos)
	case "total":
		return driver.TotalObjects(ctx, infos)
	default:
		return nil, fmt.Errorf("invalid action: %s (expected: list, total)", action)
	}
}

// ExecuteCloudVMCommand routes through JDCloud Cloud Assistant (assistant.jdcloud-api.com).
// Region must be a real VM region (cn-north-1 / cn-east-2 / ...); we resolve it
// from the host cache populated by `cloudlist` so `shell <instance-id>` works
// regardless of the session's current region setting.
func (p *Provider) ExecuteCloudVMCommand(ctx context.Context, instanceID, cmd string) (schema.CommandResult, error) {
	if osType, command, ok := vmexecspec.Parse(cmd); ok {
		if p.region == "" || p.region == "all" {
			return schema.CommandResult{}, fmt.Errorf("headless shell requires explicit region")
		}
		driver := &assistant.Driver{Client: p.apiClient, Region: p.region}
		output := driver.RunCommand(instanceID, osType, command)
		return schema.CommandResult{Output: output}, nil
	}

	if strings.HasPrefix(instanceID, "lavm-") {
		return schema.CommandResult{}, fmt.Errorf("JDCloud shell currently supports VM only")
	}
	host, ok := p.lookupHost(instanceID)
	if !ok {
		return schema.CommandResult{}, fmt.Errorf("unable to resolve instance metadata, run `cloudlist` first and retry")
	}
	command, err := base64.StdEncoding.DecodeString(cmd)
	if err != nil {
		return schema.CommandResult{}, err
	}
	driver := &assistant.Driver{Client: p.apiClient, Region: host.Region}
	output := driver.RunCommand(instanceID, host.OSType, string(command))
	return schema.CommandResult{Output: output}, nil
}

func (p *Provider) lookupHost(instanceID string) (schema.Host, bool) {
	for _, host := range vm.GetCacheHostList() {
		if host.ID == instanceID || host.HostName == instanceID {
			return host, true
		}
	}
	return schema.Host{}, false
}

func (p *Provider) newOSSDriver(region string) *oss.Driver {
	return &oss.Driver{
		Client:     p.apiClient,
		Credential: p.credential,
		Region:     region,
	}
}

func (p *Provider) bucketInfos(ctx context.Context, driver *oss.Driver, bucketName string) (map[string]string, error) {
	infos := make(map[string]string)
	bucketName = strings.TrimSpace(bucketName)
	switch {
	case bucketName == "":
		return nil, fmt.Errorf("empty bucket name")
	case bucketName == "all":
		buckets, err := driver.ListBuckets(ctx)
		if err != nil {
			return nil, err
		}
		for _, bucket := range buckets {
			bucketRegion := strings.TrimSpace(bucket.Region)
			if bucketRegion == "" {
				bucketRegion, err = driver.ResolveBucketRegion(ctx, bucket.BucketName)
				if err != nil {
					return nil, err
				}
			}
			infos[bucket.BucketName] = bucketRegion
		}
		if len(infos) == 0 {
			return nil, fmt.Errorf("no buckets found")
		}
		return infos, nil
	case p.region != "" && p.region != "all":
		infos[bucketName] = p.region
		return infos, nil
	default:
		resolved, err := driver.ResolveBucketRegion(ctx, bucketName)
		if err != nil {
			return nil, fmt.Errorf("bucket %s region not found; set region explicitly or use `list all` first", bucketName)
		}
		infos[bucketName] = resolved
		return infos, nil
	}
}
