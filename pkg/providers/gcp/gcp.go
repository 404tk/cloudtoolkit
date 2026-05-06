package gcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/auth"
	_billing "github.com/404tk/cloudtoolkit/pkg/providers/gcp/billing"
	_compute "github.com/404tk/cloudtoolkit/pkg/providers/gcp/compute"
	_dns "github.com/404tk/cloudtoolkit/pkg/providers/gcp/dns"
	_iam "github.com/404tk/cloudtoolkit/pkg/providers/gcp/iam"
	_logging "github.com/404tk/cloudtoolkit/pkg/providers/gcp/logging"
	_sqladmin "github.com/404tk/cloudtoolkit/pkg/providers/gcp/sqladmin"
	_storage "github.com/404tk/cloudtoolkit/pkg/providers/gcp/storage"
	_vmexec "github.com/404tk/cloudtoolkit/pkg/providers/gcp/vmexec"
	"github.com/404tk/cloudtoolkit/pkg/providers/internal/credverify"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/runtime/vmexecspec"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
)

// Provider is a data provider for gcp API
type Provider struct {
	cred        auth.Credential
	tokenSource *auth.TokenSource
	apiClient   *api.Client
	projects    []string
}

// ClientConfig allows callers (e.g. demo replay) to inject a custom HTTP
// client used by both the OAuth2 token source and the API client, and skip
// credential cache writes for ephemeral credentials.
type ClientConfig struct {
	HTTPClient          *http.Client
	SkipCredentialCache bool
}

// New creates a new provider client for gcp API
func New(options schema.Options) (*Provider, error) {
	return NewWithConfig(options, ClientConfig{})
}

// NewWithConfig creates a new provider client for gcp API with an injected
// HTTP transport. Real callers use New; replay/test callers feed in a mock
// HTTP client through cfg.HTTPClient.
func NewWithConfig(options schema.Options, cfg ClientConfig) (*Provider, error) {
	cred, err := auth.FromOptions(options)
	if err != nil {
		return nil, err
	}
	if err := cred.Validate(); err != nil {
		return nil, err
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = api.NewHTTPClient()
	}
	ts := auth.NewTokenSource(cred, httpClient)
	client := api.NewClient(ts, api.WithHTTPClient(httpClient))
	provider := &Provider{
		cred:        cred,
		tokenSource: ts,
		apiClient:   client,
		projects:    []string{cred.ProjectID},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := ts.Token(ctx); err != nil {
		return nil, err
	}

	if err := credverify.ForCloudlist(options, provider, cfg.SkipCredentialCache, func(context.Context) (credverify.Result, error) {
		return credverify.Result{
			Summary:     "Current project: " + cred.ProjectID,
			SessionUser: cred.ProjectID,
		}, nil
	}); err != nil {
		return nil, err
	}

	return provider, nil
}

// Name returns the name of the provider
func (p *Provider) Name() string {
	return "gcp"
}

func (p *Provider) CredentialKey(opts map[string]string) string {
	tojson, _ := base64.StdEncoding.DecodeString(opts[utils.GCPserviceAccountJSON])
	return utils.Md5Encode(string(tojson))
}

// Resources returns the provider for an resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	collector := schema.NewResourceCollector(p.Name()).
		Register("balance", func(ctx context.Context, _ *schema.Resources) {
			(&_billing.Driver{Client: p.apiClient}).QueryAccountBalance(ctx)
		}).
		Register("host", func(ctx context.Context, list *schema.Resources) {
			instanceProvider := &_compute.Driver{Projects: p.projects, Client: p.apiClient}
			computes, err := instanceProvider.GetResource(ctx)
			schema.AppendAssets(list, computes)
			list.AddError("host", err)
		}).
		Register("domain", func(ctx context.Context, list *schema.Resources) {
			cloudDNSProvider := &_dns.Driver{Projects: p.projects, Client: p.apiClient}
			domains, err := cloudDNSProvider.GetDomains(ctx)
			schema.AppendAssets(list, domains)
			list.AddError("domain", err)
		}).
		Register("account", func(ctx context.Context, list *schema.Resources) {
			saProvider := &_iam.Driver{Projects: p.projects, Client: p.apiClient}
			users, err := saProvider.ListUsers(ctx)
			schema.AppendAssets(list, users)
			list.AddError("account", err)
		}).
		Register("bucket", func(ctx context.Context, list *schema.Resources) {
			storageProvider := &_storage.Driver{Projects: p.projects, Client: p.apiClient}
			storages, err := storageProvider.GetBuckets(ctx)
			schema.AppendAssets(list, storages)
			list.AddError("bucket", err)
		}).
		Register("log", func(ctx context.Context, list *schema.Resources) {
			loggingProvider := &_logging.Driver{Client: p.apiClient, Projects: p.projects}
			logs, err := loggingProvider.GetLogs(ctx)
			schema.AppendAssets(list, logs)
			list.AddError("log", err)
		}).
		Register("database", func(ctx context.Context, list *schema.Resources) {
			sqlProvider := &_sqladmin.Driver{Client: p.apiClient, Projects: p.projects}
			dbs, err := sqlProvider.GetDatabases(ctx)
			schema.AppendAssets(list, dbs)
			list.AddError("database", err)
		})

	return collector.Collect(ctx, env.From(ctx).Cloudlist)
}

// BucketDump implements schema.BucketManager for GCP via GCS object listing.
func (p *Provider) BucketDump(ctx context.Context, action, bucketName string) ([]schema.BucketResult, error) {
	driver := &_storage.Driver{Projects: p.projects, Client: p.apiClient}
	infos := make(map[string]string)
	if bucketName == "all" {
		buckets, err := driver.GetBuckets(context.Background())
		if err != nil {
			return nil, fmt.Errorf("list buckets: %w", err)
		}
		for _, b := range buckets {
			infos[b.BucketName] = b.Region
		}
	} else {
		infos[bucketName] = ""
	}
	switch action {
	case "list":
		return driver.ListObjects(ctx, infos)
	// case "total":
	// return driver.TotalObjects(ctx, infos)
	default:
		return nil, fmt.Errorf("invalid action: %s (expected: list, total)", action)
	}
}

// BucketACL implements schema.BucketACLManager for GCP. The GCS analogue of
// "public" / "private" is the bucket IAM policy granting `allUsers` the
// objectViewer role.
func (p *Provider) BucketACL(ctx context.Context, action, container, level string) (schema.BucketACLResult, error) {
	driver := &_storage.Driver{Projects: p.projects, Client: p.apiClient}
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
		result.Level = "Private"
		result.Message = fmt.Sprintf("bucket %s reverted to private", container)
		return result, nil
	}
	return result, fmt.Errorf("gcp: unsupported bucket-acl action %q", action)
}

// UserManagement implements schema.IAMManager for GCP. Cloud Identity user
// lifecycle requires a paid Google Workspace tenant; the practical
// CSPM-detectable lever for non-Workspace projects is service account
// enable/disable. `username` is the SA email (or short name), `password` is
// ignored (SA accounts have no password).
func (p *Provider) UserManagement(action, username, _ string) (schema.IAMResult, error) {
	driver := &_iam.Driver{Projects: p.projects, Client: p.apiClient}
	ctx := context.Background()
	switch action {
	case "add":
		return driver.AddUser(ctx, username)
	case "del":
		return driver.DelUser(ctx, username)
	default:
		return schema.IAMResult{}, fmt.Errorf("invalid action: %s (expected: add, del)", action)
	}
}

// ExecuteCloudVMCommand implements schema.VMExecutor for GCP via the metadata
// startup-script + reboot path. See pkg/providers/gcp/vmexec/vmexec.go for the
// rationale (PLAN.md decision T2.2/Task 10). Output is not captured —
// startup-script stdout goes to the serial console.
//
// `cmd` arrives in one of two encodings: the headless `__ctk_headless_sh__:`
// vmexec spec (which carries an explicit osType), or a bare base64 string from
// the REPL shell loop. Windows targets are rejected here — the startup-script
// path only runs Linux bash; a Windows-equivalent (`windows-startup-script-*`
// metadata + Cloud-Init) is out of scope until separately validated.
func (p *Provider) ExecuteCloudVMCommand(ctx context.Context, instanceID, cmd string) (schema.CommandResult, error) {
	driver := &_vmexec.Driver{Projects: p.projects, Client: p.apiClient}
	if osType, command, ok := vmexecspec.Parse(cmd); ok {
		if osType == "windows" {
			return schema.CommandResult{}, fmt.Errorf("gcp vmexec: windows targets are not supported on the startup-script path")
		}
		return driver.Execute(ctx, instanceID, command)
	}
	command, err := base64.StdEncoding.DecodeString(cmd)
	if err != nil {
		return schema.CommandResult{}, err
	}
	return driver.Execute(ctx, instanceID, strings.TrimSpace(string(command)))
}

// RoleBinding implements schema.RoleBindingManager for GCP project-level IAM
// bindings. The scope argument is the project ID; an empty value falls back
// to the credential's project.
func (p *Provider) RoleBinding(ctx context.Context, action, principal, role, scope string) (schema.RoleBindingResult, error) {
	driver := &_iam.Driver{Projects: p.projects, Client: p.apiClient}
	project := strings.TrimSpace(scope)
	if project == "" && len(p.projects) > 0 {
		project = p.projects[0]
	}
	if project == "" {
		return schema.RoleBindingResult{Action: action}, fmt.Errorf("gcp: no project configured for role binding")
	}
	result := schema.RoleBindingResult{
		Action:    action,
		Principal: principal,
		Role:      role,
		Scope:     project,
	}
	switch action {
	case "list":
		policy, err := driver.GetProjectIamPolicy(ctx, project)
		if err != nil {
			return result, err
		}
		for _, b := range policy.Bindings {
			for _, member := range b.Members {
				if principal != "" && !strings.EqualFold(member, principal) {
					continue
				}
				result.Bindings = append(result.Bindings, schema.RoleBinding{
					Principal: member,
					Role:      b.Role,
					Scope:     project,
				})
			}
		}
		result.Message = fmt.Sprintf("%d role bindings on project %s", len(result.Bindings), project)
		return result, nil
	case "add":
		if _, err := driver.AddBinding(ctx, project, role, principal); err != nil {
			return result, err
		}
		result.Message = fmt.Sprintf("bound %s to %s on project %s", principal, role, project)
		return result, nil
	case "del":
		if _, err := driver.RemoveBinding(ctx, project, role, principal); err != nil {
			return result, err
		}
		result.Message = fmt.Sprintf("removed %s from %s on project %s", principal, role, project)
		return result, nil
	}
	return result, fmt.Errorf("gcp: unsupported role-binding action %q", action)
}

// IAMCredential implements schema.IAMCredentialManager. GCP currently maps the
// generic capability to service-account key lifecycle operations.
func (p *Provider) IAMCredential(ctx context.Context, action, principal, credentialID string) (schema.IAMCredentialResult, error) {
	driver := &_iam.Driver{Projects: p.projects, Client: p.apiClient}
	if len(p.projects) == 0 || p.projects[0] == "" {
		return schema.IAMCredentialResult{Action: action}, fmt.Errorf("gcp: no project configured for service account key")
	}
	project := p.projects[0]
	result := schema.IAMCredentialResult{
		Action:       action,
		Principal:    principal,
		CredentialID: credentialID,
	}
	switch action {
	case "list":
		keys, err := driver.ListKeys(ctx, project, principal)
		if err != nil {
			return result, err
		}
		for _, k := range keys {
			result.Credentials = append(result.Credentials, schema.IAMCredential{
				CredentialID:   _iam.KeyShortID(k.Name),
				CredentialType: k.KeyType,
				ValidAfter:     k.ValidAfterTime,
				ValidBefore:    k.ValidBeforeTime,
			})
		}
		result.Message = fmt.Sprintf("%d credentials on %s", len(result.Credentials), principal)
		return result, nil
	case "create":
		key, err := driver.CreateKey(ctx, project, principal)
		if err != nil {
			return result, err
		}
		result.CredentialID = _iam.KeyShortID(key.Name)
		result.CredentialData = key.PrivateKeyData
		result.Message = fmt.Sprintf("minted credential %s for %s", result.CredentialID, principal)
		return result, nil
	case "delete":
		if err := driver.DeleteKey(ctx, project, principal, credentialID); err != nil {
			return result, err
		}
		result.Message = fmt.Sprintf("revoked credential %s on %s", credentialID, principal)
		return result, nil
	}
	return result, fmt.Errorf("gcp: unsupported iam-credential action %q", action)
}

// EventDump implements schema.EventReader for GCP Cloud Audit Logs via Cloud
// Logging `entries:list`. Action `dump` lists recent audit entries scoped to
// the provider's project; `whitelist` is unsupported because Cloud Audit
// Logs are read-only.
func (p *Provider) EventDump(ctx context.Context, action, args string) (schema.EventActionResult, error) {
	driver := &_logging.Driver{Client: p.apiClient, Projects: p.projects}
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

// DBManagement implements schema.DBManager for GCP Cloud SQL. `useradd` /
// `userdel` invoke the Cloud SQL Admin user APIs.
func (p *Provider) DBManagement(ctx context.Context, action, instanceID string) (schema.DatabaseActionResult, error) {
	driver := &_sqladmin.Driver{Client: p.apiClient, Projects: p.projects}
	switch action {
	case "useradd":
		return driver.CreateAccount(ctx, instanceID)
	case "userdel":
		return driver.DeleteAccount(ctx, instanceID)
	default:
		return schema.DatabaseActionResult{}, fmt.Errorf("invalid action: %s (expected: useradd, userdel)", action)
	}
}
