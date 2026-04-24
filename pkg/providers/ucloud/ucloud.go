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
	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/udb"
	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/udns"
	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/ufile"
	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/uhost"
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
}

// New creates a new provider client for UCloud APIs.
func New(options schema.Options) (*Provider, error) {
	credential, err := ucloudauth.FromOptions(options)
	if err != nil {
		return nil, err
	}
	if err := credential.Validate(); err != nil {
		return nil, err
	}

	accountClient := api.NewClient(credential)
	user, err := currentUser(accountClient)
	if err != nil {
		return nil, err
	}

	projects, err := projectList(accountClient)
	if err != nil {
		return nil, err
	}

	projectID, projectName, err := resolveProject(projects, strings.TrimSpace(options[utils.ProjectID]))
	if err != nil {
		return nil, err
	}

	region := normalizeRegion(options[utils.Region])
	regions, err := resolveRegions(accountClient, region)
	if err != nil {
		return nil, err
	}

	if options != nil && strings.TrimSpace(options[utils.ProjectID]) == "" && projectID != "" {
		options[utils.ProjectID] = projectID
	}

	if strings.TrimSpace(options[utils.Payload]) == "cloudlist" {
		display := displayCurrentUser(user)
		if display != "" {
			logger.Warning("Current user:", display)
		} else {
			display = "<none>"
		}
		if pj := displayCurrentProject(projectID, projectName); pj != "" {
			logger.Warning("Current project:", pj)
		}
		cache.Cfg.CredInsert(display, options)
	}

	return &Provider{
		credential: credential,
		region:     region,
		projectID:  projectID,
		regions:    regions,
	}, nil
}

// Name returns the name of the provider.
func (p *Provider) Name() string {
	return "ucloud"
}

// Resources returns cloud assets for the asset inventory payload.
func (p *Provider) Resources(ctx context.Context) (schema.Resources, error) {
	list := schema.NewResources()
	list.Provider = p.Name()

	for _, product := range utils.Cloudlist {
		switch product {
		case "balance":
			(&billing.Driver{
				Credential: p.credential,
				ProjectID:  p.projectID,
			}).QueryAccountBalance(ctx)
		case "host":
			d := &uhost.Driver{
				Credential: p.credential,
				ProjectID:  p.projectID,
				Regions:    p.regions,
			}
			hosts, err := d.GetResource(ctx)
			schema.AppendAssets(&list, hosts)
			list.AddError("host", err)
		case "domain":
			d := &udns.Driver{
				Credential: p.credential,
				ProjectID:  p.projectID,
				Regions:    p.regions,
			}
			domains, err := d.GetDomains(ctx)
			schema.AppendAssets(&list, domains)
			list.AddError("domain", err)
		case "database":
			d := &udb.Driver{
				Credential: p.credential,
				ProjectID:  p.projectID,
				Regions:    p.regions,
			}
			databases, err := d.GetDatabases(ctx)
			schema.AppendAssets(&list, databases)
			list.AddError("database", err)
		case "bucket":
			d := &ufile.Driver{
				Credential: p.credential,
				ProjectID:  p.projectID,
				Region:     p.region,
			}
			buckets, err := d.GetBuckets(ctx)
			schema.AppendAssets(&list, buckets)
			list.AddError("bucket", err)
		case "account", "sms", "log":
		default:
		}
	}

	return list, list.Err()
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
			if strings.TrimSpace(project.ProjectID) == configured {
				return strings.TrimSpace(project.ProjectID), strings.TrimSpace(project.ProjectName), nil
			}
		}
		return "", "", fmt.Errorf("ucloud projectId not found or inaccessible: %s", configured)
	}

	for _, project := range projects {
		if project.IsDefault {
			return strings.TrimSpace(project.ProjectID), strings.TrimSpace(project.ProjectName), nil
		}
	}

	switch len(projects) {
	case 0:
		return "", "", errors.New("no accessible UCloud project found")
	case 1:
		return strings.TrimSpace(projects[0].ProjectID), strings.TrimSpace(projects[0].ProjectName), nil
	default:
		return "", "", errors.New("multiple accessible UCloud projects found; set projectId explicitly")
	}
}

func resolveRegions(client *api.Client, requested string) ([]string, error) {
	if !strings.EqualFold(strings.TrimSpace(requested), "all") {
		return []string{strings.TrimSpace(requested)}, nil
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
		return ""
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
