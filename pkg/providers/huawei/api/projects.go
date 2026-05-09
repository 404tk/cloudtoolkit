package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type ProjectNotFoundError struct {
	Region string
}

func (e *ProjectNotFoundError) Error() string {
	return "project not found"
}

func IsProjectNotFound(err error) bool {
	var target *ProjectNotFoundError
	return errors.As(err, &target)
}

type ProjectCatalog struct {
	byRegion map[string]string
}

// NewProjectCatalog indexes IAM projects by region name, preferring projects
// in domainID when duplicate names are present.
func NewProjectCatalog(projects []IAMProject, domainID string) *ProjectCatalog {
	domainID = strings.TrimSpace(domainID)
	matched := make(map[string]string, len(projects))
	fallback := make(map[string]string)
	for _, project := range projects {
		region := strings.TrimSpace(project.Name)
		projectID := strings.TrimSpace(project.ID)
		if region == "" || projectID == "" {
			continue
		}
		if domainID == "" || strings.TrimSpace(project.DomainID) == domainID {
			if matched[region] == "" {
				matched[region] = projectID
			}
			continue
		}
		if fallback[region] == "" {
			fallback[region] = projectID
		}
	}
	for region, projectID := range fallback {
		if matched[region] == "" {
			matched[region] = projectID
		}
	}
	return &ProjectCatalog{byRegion: matched}
}

// ProjectID returns the project ID for region when the catalog contains it.
func (c *ProjectCatalog) ProjectID(region string) (string, bool) {
	if c == nil {
		return "", false
	}
	projectID := strings.TrimSpace(c.byRegion[strings.TrimSpace(region)])
	return projectID, projectID != ""
}

// FilterRegions keeps only regions that exist in the catalog.
func (c *ProjectCatalog) FilterRegions(regions []string) []string {
	if c == nil {
		return append([]string(nil), regions...)
	}
	filtered := make([]string, 0, len(regions))
	for _, region := range regions {
		if _, ok := c.ProjectID(region); ok {
			filtered = append(filtered, region)
		}
	}
	return filtered
}

// ListAccessibleProjectCatalog fetches the project list visible to the current
// credential and indexes it by region name.
func ListAccessibleProjectCatalog(ctx context.Context, client *Client, domainID string) (*ProjectCatalog, error) {
	if client == nil {
		return nil, fmt.Errorf("huawei project catalog: nil client")
	}

	controlPlaneRegion := strings.TrimSpace(client.credential.Region)
	if controlPlaneRegion == "" || controlPlaneRegion == "all" {
		return nil, fmt.Errorf("huawei project catalog: unresolved control plane region %q", controlPlaneRegion)
	}

	var resp ListProjectsResponse
	if err := client.DoJSON(ctx, Request{
		Service:    "iam",
		Region:     controlPlaneRegion,
		Intl:       client.credential.Intl,
		Method:     http.MethodGet,
		Path:       "/v3/auth/projects",
		Idempotent: true,
	}, &resp); err != nil {
		return nil, err
	}
	return NewProjectCatalog(resp.Projects, domainID), nil
}

// ResolveProjectID maps a region name to its project ID via the IAM control
// plane. The client must be configured with a concrete control-plane region.
func ResolveProjectID(ctx context.Context, client *Client, domainID, targetRegion string) (string, error) {
	if client == nil {
		return "", fmt.Errorf("huawei project resolver: nil client")
	}

	controlPlaneRegion := strings.TrimSpace(client.credential.Region)
	if controlPlaneRegion == "" || controlPlaneRegion == "all" {
		return "", fmt.Errorf("huawei project resolver: unresolved control plane region %q", controlPlaneRegion)
	}

	targetRegion = strings.TrimSpace(targetRegion)
	if targetRegion == "" {
		return "", fmt.Errorf("huawei project resolver: empty target region")
	}

	query := url.Values{}
	query.Set("name", targetRegion)
	domainID = strings.TrimSpace(domainID)

	var resp ListProjectsResponse
	if err := client.DoJSON(ctx, Request{
		Service:    "iam",
		Region:     controlPlaneRegion,
		Intl:       client.credential.Intl,
		Method:     http.MethodGet,
		Path:       "/v3/projects",
		Query:      query,
		Idempotent: true,
	}, &resp); err != nil {
		return "", err
	}

	fallback := ""
	for _, project := range resp.Projects {
		if strings.TrimSpace(project.Name) != targetRegion {
			continue
		}
		projectID := strings.TrimSpace(project.ID)
		if fallback == "" {
			fallback = projectID
		}
		if domainID == "" || strings.TrimSpace(project.DomainID) == domainID {
			return projectID, nil
		}
	}
	if fallback != "" {
		return fallback, nil
	}
	return "", &ProjectNotFoundError{Region: targetRegion}
}
