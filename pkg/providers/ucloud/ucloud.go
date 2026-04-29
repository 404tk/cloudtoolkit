package ucloud

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	ucloudauth "github.com/404tk/cloudtoolkit/pkg/providers/ucloud/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/billing"
	_iam "github.com/404tk/cloudtoolkit/pkg/providers/ucloud/iam"
	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/udb"
	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/udns"
	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/ufile"
	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/uhost"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/cache"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Provider struct {
	credential ucloudauth.Credential
	region     string
	projectID  string
	regions    []string
	apiOptions []api.Option
	apiClient  *api.Client
}

// ClientConfig allows callers (e.g. demo replay) to inject custom api.Option
// values and skip credential cache writes for ephemeral credentials.
type ClientConfig struct {
	APIOptions          []api.Option
	SkipCredentialCache bool
}

// New creates a new provider client for UCloud APIs.
func New(options schema.Options) (*Provider, error) {
	return NewWithConfig(options, ClientConfig{})
}

// NewWithConfig creates a new provider client for UCloud APIs with injected
// transport options. Real callers use New; replay/test callers feed in a
// mock HTTP client through cfg.APIOptions.
func NewWithConfig(options schema.Options, cfg ClientConfig) (*Provider, error) {
	credential, err := ucloudauth.FromOptions(options)
	if err != nil {
		return nil, err
	}
	if err := credential.Validate(); err != nil {
		return nil, err
	}

	apiOptions := append([]api.Option(nil), cfg.APIOptions...)
	accountClient := api.NewClient(credential, apiOptions...)
	user, err := currentUser(accountClient)
	if err != nil {
		return nil, err
	}

	projects, err := projectList(accountClient)
	if err != nil {
		return nil, err
	}

	projectOption := strings.TrimSpace(options[utils.ProjectID])
	projectID, projectName, err := resolveProject(projects, projectOption)
	if err != nil {
		return nil, err
	}

	region := normalizeRegion(options[utils.Region])
	regions, err := resolveRegions(accountClient, region)
	if err != nil {
		return nil, err
	}

	if options != nil && projectOption == "" && projectID != "" {
		options[utils.ProjectID] = projectID
	}
	provider := &Provider{
		credential: credential,
		region:     region,
		projectID:  projectID,
		regions:    regions,
		apiOptions: apiOptions,
		apiClient:  accountClient,
	}

	payload := strings.TrimSpace(options[utils.Payload])
	if payload == "cloudlist" {
		display := displayCurrentUser(user)
		logger.Warning("Current user:", display)
		if pj := displayCurrentProject(projectID, projectName); pj != "" {
			logger.Warning("Current project:", pj)
		}
		if !cfg.SkipCredentialCache {
			cache.Cfg.CredInsert(display, provider, options)
		}
	}

	return provider, nil
}

// Name returns the name of the provider.
func (p *Provider) Name() string {
	return "ucloud"
}

func (p *Provider) newClient() *api.Client {
	if p.apiClient != nil {
		return p.apiClient
	}
	return api.NewClient(p.credential, p.apiOptions...)
}

// Resources returns cloud assets for the asset inventory payload.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	collector := schema.NewResourceCollector(p.Name()).
		Register("balance", func(ctx context.Context, _ *schema.Resources) {
			(&billing.Driver{
				Credential: p.credential,
				Client:     p.newClient(),
				ProjectID:  p.projectID,
			}).QueryAccountBalance(ctx)
		}).
		Register("host", func(ctx context.Context, list *schema.Resources) {
			d := &uhost.Driver{
				Credential: p.credential,
				Client:     p.newClient(),
				ProjectID:  p.projectID,
				Regions:    p.regions,
			}
			hosts, err := d.GetResource(ctx)
			schema.AppendAssets(list, hosts)
			list.AddError("host", err)
		}).
		Register("domain", func(ctx context.Context, list *schema.Resources) {
			d := &udns.Driver{
				Credential: p.credential,
				Client:     p.newClient(),
				ProjectID:  p.projectID,
				Regions:    p.regions,
			}
			domains, err := d.GetDomains(ctx)
			schema.AppendAssets(list, domains)
			list.AddError("domain", err)
		}).
		Register("database", func(ctx context.Context, list *schema.Resources) {
			d := &udb.Driver{
				Credential: p.credential,
				Client:     p.newClient(),
				ProjectID:  p.projectID,
				Regions:    p.regions,
			}
			databases, err := d.GetDatabases(ctx)
			schema.AppendAssets(list, databases)
			list.AddError("database", err)
		}).
		Register("bucket", func(ctx context.Context, list *schema.Resources) {
			d := &ufile.Driver{
				Credential: p.credential,
				Client:     p.newClient(),
				ProjectID:  p.projectID,
				Region:     p.region,
			}
			buckets, err := d.GetBuckets(ctx)
			schema.AppendAssets(list, buckets)
			list.AddError("bucket", err)
		}).
		Register("account", func(ctx context.Context, list *schema.Resources) {
			d := &_iam.Driver{Credential: p.credential, Client: p.newClient()}
			users, err := d.ListUsers(ctx)
			schema.AppendAssets(list, users)
			list.AddError("account", err)
		})

	return collector.Collect(ctx, env.From(ctx).Cloudlist)
}

func (p *Provider) UserManagement(action, username, password string) (schema.IAMResult, error) {
	driver := &_iam.Driver{
		Credential: p.credential,
		Client:     p.newClient(),
		ProjectID:  p.projectID,
		UserName:   username,
		Password:   password,
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

func normalizeRegion(region string) string {
	region = strings.TrimSpace(region)
	if region == "" {
		return "all"
	}
	return region
}

func currentUser(client *api.Client) (api.UserInfo, error) {
	var resp api.GetUserInfoResponse
	err := client.Do(context.Background(), api.Request{Action: "GetUserInfo"}, &resp)
	if err != nil {
		return api.UserInfo{}, err
	}
	if len(resp.DataSet) == 0 {
		return api.UserInfo{}, errors.New("ucloud GetUserInfo returned no user data")
	}
	return resp.DataSet[0], nil
}

func projectList(client *api.Client) ([]api.ProjectListInfo, error) {
	var resp api.GetProjectListResponse
	err := client.Do(context.Background(), api.Request{Action: "GetProjectList"}, &resp)
	if err != nil {
		return nil, err
	}
	return resp.ProjectSet, nil
}

func resolveProject(projects []api.ProjectListInfo, configured string) (string, string, error) {
	configured = strings.TrimSpace(configured)
	if configured != "" {
		for _, project := range projects {
			projectID, projectName := trimProject(project)
			if projectID == configured {
				return projectID, projectName, nil
			}
		}
		return "", "", fmt.Errorf("ucloud projectId not found or inaccessible: %s", configured)
	}

	for _, project := range projects {
		if project.IsDefault {
			projectID, projectName := trimProject(project)
			return projectID, projectName, nil
		}
	}

	switch len(projects) {
	case 0:
		return "", "", errors.New("no accessible UCloud project found")
	case 1:
		projectID, projectName := trimProject(projects[0])
		return projectID, projectName, nil
	default:
		return "", "", errors.New("multiple accessible UCloud projects found; set projectId explicitly")
	}
}

func resolveRegions(client *api.Client, requested string) ([]string, error) {
	requested = strings.TrimSpace(requested)
	if !strings.EqualFold(requested, "all") {
		return []string{requested}, nil
	}

	var resp api.GetRegionResponse
	err := client.Do(context.Background(), api.Request{Action: "GetRegion"}, &resp)
	if err != nil {
		return nil, err
	}

	regions := make([]string, 0, len(resp.Regions))
	seen := make(map[string]struct{}, len(resp.Regions))
	for _, item := range resp.Regions {
		region := strings.TrimSpace(item.Region)
		if region == "" {
			continue
		}
		if _, ok := seen[region]; ok {
			continue
		}
		seen[region] = struct{}{}
		regions = append(regions, region)
	}
	return regions, nil
}

func displayCurrentUser(user api.UserInfo) string {
	name := strings.TrimSpace(user.UserName)
	email := strings.TrimSpace(user.UserEmail)
	switch {
	case name != "" && email != "":
		return fmt.Sprintf("%s (%s)", name, email)
	case name != "":
		return name
	case email != "":
		return email
	case user.UserID > 0:
		return "user-" + strconv.Itoa(user.UserID)
	default:
		return "<none>"
	}
}

func displayCurrentProject(projectID, projectName string) string {
	projectID = strings.TrimSpace(projectID)
	projectName = strings.TrimSpace(projectName)
	switch {
	case projectName != "" && projectID != "" && projectName != projectID:
		return fmt.Sprintf("%s (%s)", projectName, projectID)
	case projectName != "":
		return projectName
	default:
		return projectID
	}
}

func trimProject(project api.ProjectListInfo) (string, string) {
	return strings.TrimSpace(project.ProjectID), strings.TrimSpace(project.ProjectName)
}
