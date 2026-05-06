package azure

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	azapi "github.com/404tk/cloudtoolkit/pkg/providers/azure/api"
	azauth "github.com/404tk/cloudtoolkit/pkg/providers/azure/auth"
	azcloud "github.com/404tk/cloudtoolkit/pkg/providers/azure/cloud"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/compute"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/graph"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/insights"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/rbac"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/sqldb"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/storage"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/runtime/vmexecspec"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

// Provider is a data provider for Azure ARM APIs.
type Provider struct {
	cred             azauth.Credential
	endpoints        azcloud.Endpoints
	tokenSource      *azauth.TokenSource
	apiClient        *azapi.Client
	graphTokenSource *azauth.TokenSource
	graphHTTPClient  *http.Client
	subscriptionIDs  []string
}

// ClientConfig allows callers (e.g. demo replay) to inject a custom HTTP
// client used by both the OAuth2 token source and the ARM API client, and
// skip credential cache writes for ephemeral credentials.
type ClientConfig struct {
	HTTPClient          *http.Client
	SkipCredentialCache bool
}

// New creates a new provider client for Azure API.
func New(options schema.Options) (*Provider, error) {
	return NewWithConfig(options, ClientConfig{})
}

// NewWithConfig creates a new provider client for Azure API with an injected
// HTTP transport. Real callers use New; replay/test callers feed in a mock
// HTTP client through cfg.HTTPClient.
func NewWithConfig(options schema.Options, cfg ClientConfig) (*Provider, error) {
	cred, err := azauth.FromOptions(options)
	if err != nil {
		return nil, err
	}
	if err := cred.Validate(); err != nil {
		return nil, err
	}

	endpoints := azcloud.For(cred.Cloud)
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = azapi.NewHTTPClient()
	}
	tokenSource := azauth.NewTokenSource(cred, httpClient)
	client := azapi.NewClient(tokenSource, endpoints, azapi.WithHTTPClient(httpClient))
	graphScope := cred.Cloud.MicrosoftGraphEndpoint() + ".default"
	graphTokenSource := azauth.NewTokenSourceForScope(cred, httpClient, graphScope)
	provider := &Provider{
		cred:             cred,
		endpoints:        endpoints,
		tokenSource:      tokenSource,
		apiClient:        client,
		graphTokenSource: graphTokenSource,
		graphHTTPClient:  httpClient,
	}

	subscriptionIDs := make([]string, 0, 1)
	if cred.SubscriptionID != "" {
		subscriptionIDs = append(subscriptionIDs, cred.SubscriptionID)
	} else {
		pager := azapi.NewPager[azapi.Subscription](client, azapi.Request{
			Method:     http.MethodGet,
			Path:       "/subscriptions",
			Query:      url.Values{"api-version": {azapi.SubscriptionsAPIVersion}},
			Idempotent: true,
		})
		allSubscriptions, err := pager.All(context.Background())
		if err != nil {
			return nil, err
		}
		payload, _ := options.GetMetadata(utils.Payload)
		for _, sub := range allSubscriptions {
			if payload == "cloudlist" {
				logger.Warning(fmt.Sprintf("Found Subscription: %s(%s)", sub.DisplayName, sub.SubscriptionID))
				if !cfg.SkipCredentialCache {
					cache.Cfg.CredInsert(sub.DisplayName, provider, options)
				}
			}
			if sub.SubscriptionID != "" {
				subscriptionIDs = append(subscriptionIDs, sub.SubscriptionID)
			}
		}
	}

	if len(subscriptionIDs) == 0 || subscriptionIDs[0] == "" {
		return nil, errors.New("no subscription found")
	}

	provider.subscriptionIDs = subscriptionIDs
	return provider, nil
}

// Name returns the name of the provider.
func (p *Provider) Name() string {
	return "azure"
}

func (p *Provider) CredentialKey(opts map[string]string) string {
	return opts[utils.AzureClientId]
}

// Resources returns the provider for a resource deployment source.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	collector := schema.NewResourceCollector(p.Name()).
		Register("host", func(ctx context.Context, list *schema.Resources) {
			vmProvider := &compute.Driver{
				Client:          p.apiClient,
				SubscriptionIDs: p.subscriptionIDs,
			}
			hosts, err := vmProvider.GetResource(ctx)
			schema.AppendAssets(list, hosts)
			list.AddError("host", err)
		}).
		Register("bucket", func(ctx context.Context, list *schema.Resources) {
			storageProvider := &storage.Driver{
				Client:          p.apiClient,
				SubscriptionIDs: p.subscriptionIDs,
			}
			storages, err := storageProvider.GetStorages(ctx)
			schema.AppendAssets(list, storages)
			list.AddError("bucket", err)
		}).
		Register("account", func(ctx context.Context, list *schema.Resources) {
			graphClient := graph.NewClient(p.graphTokenSource, p.graphHTTPClient, p.cred.Cloud.MicrosoftGraphEndpoint())
			users, err := graphClient.ListUsers(ctx)
			accounts := make([]schema.User, 0, len(users))
			for _, user := range users {
				userName := firstNonEmpty(user.UserPrincipalName, user.DisplayName, user.ID)
				if userName == "" {
					continue
				}
				account := schema.User{
					UserName:    userName,
					UserId:      user.ID,
					EnableLogin: user.AccountEnabled,
					CreateTime:  user.CreatedDateTime,
				}
				if user.SignInActivity != nil {
					account.LastLogin = user.SignInActivity.LastSignInDateTime
				}
				accounts = append(accounts, account)
			}
			schema.AppendAssets(list, accounts)
			list.AddError("account", err)
		})

	return collector.Collect(ctx, env.From(ctx).Cloudlist)
}

// UserManagement implements schema.IAMManager for Azure via Microsoft Graph
// `users` POST/DELETE. Action `add` provisions an Azure AD user with the
// supplied initial password; `del` revokes it.
func (p *Provider) UserManagement(action, username, password string) (schema.IAMResult, error) {
	graphClient := graph.NewClient(p.graphTokenSource, p.graphHTTPClient, p.cred.Cloud.MicrosoftGraphEndpoint())
	ctx := context.Background()
	upn := strings.TrimSpace(username)
	if !strings.Contains(upn, "@") {
		return schema.IAMResult{}, fmt.Errorf("azure: username must be a full userPrincipalName (for example user@example.com)")
	}
	switch action {
	case "add":
		user, err := graphClient.CreateUser(ctx, upn, upn, password)
		if err != nil {
			return schema.IAMResult{}, err
		}
		return schema.IAMResult{
			Action:   "add",
			Username: user.UserPrincipalName,
			Password: password,
			LoginURL: "https://portal.azure.com",
			Message:  "Azure AD user created",
		}, nil
	case "del":
		if err := graphClient.DeleteUser(ctx, upn); err != nil {
			return schema.IAMResult{}, err
		}
		return schema.IAMResult{
			Action:   "del",
			Username: upn,
			Message:  upn + " user delete completed.",
		}, nil
	}
	return schema.IAMResult{}, fmt.Errorf("invalid action: %s (expected: add, del)", action)
}

// RoleBinding implements schema.RoleBindingManager. It dispatches list / add /
// del actions to the rbac driver. An empty scope falls back to the first
// configured subscription.
func (p *Provider) RoleBinding(ctx context.Context, action, principal, role, scope string) (schema.RoleBindingResult, error) {
	driver := &rbac.Driver{Client: p.apiClient, SubscriptionIDs: p.subscriptionIDs}
	scope = strings.TrimSpace(scope)
	if scope == "" {
		scope = driver.DefaultScope()
	}
	result := schema.RoleBindingResult{
		Action:    action,
		Principal: principal,
		Role:      role,
		Scope:     scope,
	}
	switch action {
	case "list":
		assignments, err := driver.List(ctx, scope, principal)
		if err != nil {
			return result, err
		}
		for _, a := range assignments {
			result.Bindings = append(result.Bindings, schema.RoleBinding{
				Principal:    a.Properties.PrincipalID,
				Role:         azureRoleNameFromDefinitionID(a.Properties.RoleDefinitionID),
				Scope:        firstNonEmpty(a.Properties.Scope, scope),
				AssignmentID: a.Name,
			})
		}
		result.Message = fmt.Sprintf("%d role assignments at %s", len(result.Bindings), scope)
		return result, nil
	case "add":
		assignment, err := driver.Create(ctx, scope, principal, role)
		if err != nil {
			return result, err
		}
		result.AssignmentID = assignment.Name
		result.Message = fmt.Sprintf("bound principal %s to %s at %s", principal, role, scope)
		return result, nil
	case "del":
		assignmentName, err := driver.Delete(ctx, scope, "", principal, role)
		if err != nil {
			return result, err
		}
		result.AssignmentID = assignmentName
		result.Message = fmt.Sprintf("removed assignment %s at %s", assignmentName, scope)
		return result, nil
	}
	return result, fmt.Errorf("azure: unsupported role-binding action %q", action)
}

// BucketACL implements schema.BucketACLManager.
func (p *Provider) BucketACL(ctx context.Context, action, container, level string) (schema.BucketACLResult, error) {
	driver := &storage.Driver{Client: p.apiClient, SubscriptionIDs: p.subscriptionIDs}
	result := schema.BucketACLResult{
		Action:    action,
		Container: container,
		Level:     level,
	}
	switch action {
	case "audit":
		all, err := driver.ListBlobContainers(ctx)
		if err != nil {
			return result, err
		}
		for _, c := range all {
			if container != "" && c.Name != container {
				continue
			}
			result.Containers = append(result.Containers, schema.BucketACLEntry{
				Account:   c.AccountName,
				Container: c.Name,
				Level:     c.PublicAccess,
			})
		}
		result.Message = fmt.Sprintf("%d containers audited", len(result.Containers))
		return result, nil
	case "expose":
		target, err := driver.FindContainer(ctx, container)
		if err != nil {
			return result, err
		}
		desired := level
		if strings.TrimSpace(desired) == "" {
			desired = "Blob"
		}
		if err := driver.SetContainerACL(ctx, target.Subscription, target.ResourceGroup, target.AccountName, target.Name, desired); err != nil {
			return result, err
		}
		applied, _ := driver.GetContainerACL(ctx, target.Subscription, target.ResourceGroup, target.AccountName, target.Name)
		result.Level = applied
		result.Message = fmt.Sprintf("container %s public access set to %s", container, applied)
		return result, nil
	case "unexpose":
		target, err := driver.FindContainer(ctx, container)
		if err != nil {
			return result, err
		}
		if err := driver.SetContainerACL(ctx, target.Subscription, target.ResourceGroup, target.AccountName, target.Name, "None"); err != nil {
			return result, err
		}
		result.Level = "None"
		result.Message = fmt.Sprintf("container %s reverted to private", container)
		return result, nil
	}
	return result, fmt.Errorf("azure: unsupported bucket-acl action %q", action)
}

// IAMCredential implements schema.IAMCredentialManager for Azure. The capability
// maps to Microsoft Graph application password credential lifecycle: list /
// addPassword / removePassword. `principal` is the Azure AD application ID
// (objectId or appId); `credentialID` is the password keyId for delete.
func (p *Provider) IAMCredential(ctx context.Context, action, principal, credentialID string) (schema.IAMCredentialResult, error) {
	graphClient := graph.NewClient(p.graphTokenSource, p.graphHTTPClient, p.cred.Cloud.MicrosoftGraphEndpoint())
	result := schema.IAMCredentialResult{
		Action:       action,
		Principal:    principal,
		CredentialID: credentialID,
	}
	switch action {
	case "list":
		app, err := graphClient.ListPasswordCredentials(ctx, principal)
		if err != nil {
			return result, err
		}
		for _, pc := range app.PasswordCredentials {
			result.Credentials = append(result.Credentials, schema.IAMCredential{
				CredentialID: pc.KeyID,
				ValidAfter:   pc.StartDateTime,
				ValidBefore:  pc.EndDateTime,
			})
		}
		result.Message = fmt.Sprintf("%d password credentials on application %s", len(result.Credentials), app.DisplayName)
		return result, nil
	case "create":
		pc, err := graphClient.AddPassword(ctx, principal, "ctk validation secret")
		if err != nil {
			return result, err
		}
		result.CredentialID = pc.KeyID
		result.CredentialData = pc.SecretText
		result.Message = fmt.Sprintf("minted password %s on application %s", pc.KeyID, principal)
		return result, nil
	case "delete":
		if err := graphClient.RemovePassword(ctx, principal, credentialID); err != nil {
			return result, err
		}
		result.Message = fmt.Sprintf("revoked password %s on application %s", credentialID, principal)
		return result, nil
	}
	return result, fmt.Errorf("azure: unsupported iam-credential action %q", action)
}

// EventDump implements schema.EventReader for Azure Activity Log. The `dump`
// action lists recent management-plane events; `whitelist` is unsupported
// because Activity Log is read-only.
func (p *Provider) EventDump(ctx context.Context, action, args string) (schema.EventActionResult, error) {
	driver := &insights.Driver{Client: p.apiClient, SubscriptionIDs: p.subscriptionIDs}
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

// DBManagement implements schema.DBManager for Azure SQL by rotating the
// server administratorLoginPassword. Azure SQL has no native "user" API at
// ARM (T-SQL is required); rotating the admin password is the closest
// CSPM-detectable management-plane signal. instanceID is parsed as
// `<resourceGroup>/<serverName>`.
func (p *Provider) DBManagement(ctx context.Context, action, instanceID string) (schema.DatabaseActionResult, error) {
	driver := &sqldb.Driver{Client: p.apiClient, SubscriptionIDs: p.subscriptionIDs}
	switch action {
	case "useradd":
		return driver.CreateAccount(ctx, instanceID)
	case "userdel":
		return driver.DeleteAccount(ctx, instanceID)
	default:
		return schema.DatabaseActionResult{}, fmt.Errorf("invalid action: %s (expected: useradd, userdel)", action)
	}
}

// ExecuteCloudVMCommand routes through Microsoft.Compute virtualMachines/runCommand.
// instanceID may be a full ARM VM ID, `<subscription>/<resourceGroup>/<vmName>`,
// or the legacy `<resourceGroup>/<vmName>` shorthand. Headless `shell -t/-l`
// paths pre-encode the script; the bare `cmd` (base64) path arrives via the
// REPL shell loop.
func (p *Provider) ExecuteCloudVMCommand(ctx context.Context, instanceID, cmd string) (schema.CommandResult, error) {
	driver := &compute.Driver{Client: p.apiClient, SubscriptionIDs: p.subscriptionIDs}
	if osType, command, ok := vmexecspec.Parse(cmd); ok {
		out, err := driver.RunCommand(ctx, instanceID, osType, command)
		if err != nil {
			return schema.CommandResult{}, err
		}
		return schema.CommandResult{Output: out}, nil
	}
	command, err := base64.StdEncoding.DecodeString(cmd)
	if err != nil {
		return schema.CommandResult{}, err
	}
	out, err := driver.RunCommand(ctx, instanceID, "linux", strings.TrimSpace(string(command)))
	if err != nil {
		return schema.CommandResult{}, err
	}
	return schema.CommandResult{Output: out}, nil
}

// azureRoleNameFromDefinitionID extracts the role-definition GUID from a
// fully-qualified roleDefinitionId. The returned string is the trailing GUID;
// callers that need the human role name should resolve it via roleDefinitions.
func azureRoleNameFromDefinitionID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	idx := strings.LastIndex(id, "/")
	if idx < 0 {
		return id
	}
	return id[idx+1:]
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}
