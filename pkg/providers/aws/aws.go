package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	_api "github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	_auth "github.com/404tk/cloudtoolkit/pkg/providers/aws/auth"
	_cloudtrail "github.com/404tk/cloudtoolkit/pkg/providers/aws/cloudtrail"
	_ec2 "github.com/404tk/cloudtoolkit/pkg/providers/aws/ec2"
	_iam "github.com/404tk/cloudtoolkit/pkg/providers/aws/iam"
	_s3 "github.com/404tk/cloudtoolkit/pkg/providers/aws/s3"
	_ssm "github.com/404tk/cloudtoolkit/pkg/providers/aws/ssm"
	"github.com/404tk/cloudtoolkit/pkg/providers/internal/credverify"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/runtime/vmexecspec"
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
	return NewWithConfig(options, ClientConfig{})
}

// ClientConfig allows callers (e.g. demo replay) to inject custom api.Option
// values and skip credential cache writes for ephemeral credentials.
type ClientConfig struct {
	APIOptions          []_api.Option
	SkipCredentialCache bool
}

// NewWithConfig creates a new provider client for aws API with injected
// transport options. Real callers use New; replay/test callers feed in a
// mock HTTP client through cfg.APIOptions.
func NewWithConfig(options schema.Options, cfg ClientConfig) (*Provider, error) {
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
	apiClient := _api.NewClient(credential, cfg.APIOptions...)
	provider := &Provider{
		region:        region,
		defaultRegion: defaultRegion,
		apiClient:     apiClient,
	}

	if err := credverify.ForCloudlist(options, provider, cfg.SkipCredentialCache, func(ctx context.Context) (credverify.Result, error) {
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
			_ssm.SetCacheHostList(hosts)
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

// RoleBinding implements schema.RoleBindingManager for AWS IAM. `principal` is
// the IAM user name and `role` is the managed-policy ARN (or short name like
// "AdministratorAccess", which is expanded to the AWS-managed ARN). `scope` is
// reserved for future use; AWS user-policy attachments are global.
func (p *Provider) RoleBinding(ctx context.Context, action, principal, role, scope string) (schema.RoleBindingResult, error) {
	driver := &_iam.Driver{
		Client:        p.apiClient,
		Region:        p.region,
		DefaultRegion: p.defaultRegion,
	}
	resolvedRole := _iam.ResolvePolicyARN(role)
	result := schema.RoleBindingResult{
		Action:    action,
		Principal: principal,
		Role:      resolvedRole,
		Scope:     scope,
	}
	switch action {
	case "list":
		bindings, err := driver.ListRoleBindings(ctx, principal)
		if err != nil {
			return result, err
		}
		result.Bindings = bindings
		result.Message = fmt.Sprintf("%d managed policies attached to user %s", len(bindings), principal)
		return result, nil
	case "add":
		if err := driver.AttachPolicy(ctx, principal, resolvedRole); err != nil {
			return result, err
		}
		result.AssignmentID = resolvedRole
		result.Message = fmt.Sprintf("attached %s to user %s", resolvedRole, principal)
		return result, nil
	case "del":
		if err := driver.DetachPolicy(ctx, principal, resolvedRole); err != nil {
			return result, err
		}
		result.AssignmentID = resolvedRole
		result.Message = fmt.Sprintf("detached %s from user %s", resolvedRole, principal)
		return result, nil
	}
	return result, fmt.Errorf("aws: unsupported role-binding action %q", action)
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

// BucketACL implements schema.BucketACLManager for AWS S3. `level` accepts the
// canned S3 ACL values (private / public-read / public-read-write /
// authenticated-read / aws-exec-read) or friendly aliases resolved by
// s3.NormalizeS3ACL. The expose path also clears the bucket Public Access
// Block (best-effort) so a public canned ACL actually surfaces.
func (p *Provider) BucketACL(ctx context.Context, action, container, level string) (schema.BucketACLResult, error) {
	driver := &_s3.Driver{Client: p.apiClient, DefaultRegion: p.defaultRegion}
	result := schema.BucketACLResult{
		Action:    action,
		Container: container,
		Level:     level,
	}
	switch action {
	case "audit":
		entries, err := driver.AuditBucketACL(ctx, container)
		if err != nil {
			return result, err
		}
		result.Containers = entries
		result.Message = fmt.Sprintf("%d buckets audited", len(entries))
		return result, nil
	case "expose":
		applied, err := driver.ExposeBucket(ctx, container, level)
		if err != nil {
			return result, err
		}
		result.Level = applied
		result.Message = fmt.Sprintf("bucket %s set to %s", container, applied)
		return result, nil
	case "unexpose":
		if err := driver.UnexposeBucket(ctx, container); err != nil {
			return result, err
		}
		result.Level = _s3.S3ACLPrivate
		result.Message = fmt.Sprintf("bucket %s reverted to private", container)
		return result, nil
	}
	return result, fmt.Errorf("aws: unsupported bucket-acl action %q", action)
}

// EventDump implements schema.EventReader for AWS CloudTrail. The `dump`
// action lists recent management-event records via `LookupEvents`. CloudTrail
// is read-only — `whitelist` returns a clear unsupported error.
func (p *Provider) EventDump(ctx context.Context, action, args string) (schema.EventActionResult, error) {
	driver := &_cloudtrail.Driver{
		Client:        p.apiClient,
		Region:        p.region,
		DefaultRegion: p.defaultRegion,
	}
	switch action {
	case "dump":
		events, err := driver.DumpEvents(ctx, args)
		if err != nil {
			return schema.EventActionResult{}, err
		}
		return schema.EventActionResult{
			Action: "dump",
			Scope:  args,
			Events: events,
		}, nil
	case "whitelist":
		return driver.HandleEvents(ctx, args)
	default:
		return schema.EventActionResult{}, fmt.Errorf("invalid action: %s (expected: dump, whitelist)", action)
	}
}

// ExecuteCloudVMCommand routes through AWS Systems Manager (SSM): the
// validation flow sends a one-shot command to the target instance via
// `SendCommand` and polls `GetCommandInvocation` until the status is
// terminal. The instance must have the SSM agent running with a role that
// allows `ssm:UpdateInstanceInformation` (the default-managed-instance
// pattern) — without that the command sits in `InProgress` forever and the
// caller eventually surfaces a timeout error.
func (p *Provider) ExecuteCloudVMCommand(ctx context.Context, instanceID, cmd string) (schema.CommandResult, error) {
	if osType, command, ok := vmexecspec.Parse(cmd); ok {
		region := p.region
		if region == "" || region == "all" {
			region = p.defaultRegion
		}
		if region == "" || region == "all" {
			return schema.CommandResult{}, fmt.Errorf("headless shell requires explicit region")
		}
		driver := &_ssm.Driver{Client: p.apiClient, Region: region}
		output := driver.RunCommand(instanceID, osType, command)
		return schema.CommandResult{Output: output}, nil
	}

	region, osType, ok := (&_ssm.Driver{}).ResolveInstanceContext(instanceID)
	if !ok {
		return schema.CommandResult{}, fmt.Errorf("unable to resolve instance metadata, run `cloudlist` first and retry")
	}
	if region == "" {
		region = p.defaultRegion
	}
	command, err := base64.StdEncoding.DecodeString(cmd)
	if err != nil {
		return schema.CommandResult{}, err
	}
	driver := &_ssm.Driver{Client: p.apiClient, Region: region}
	output := driver.RunCommand(instanceID, osType, strings.TrimSpace(string(command)))
	return schema.CommandResult{Output: output}, nil
}
