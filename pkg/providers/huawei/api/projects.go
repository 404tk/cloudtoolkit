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
	if e == nil {
		return "project not found"
	}
	return fmt.Sprintf("project not found for region %s", strings.TrimSpace(e.Region))
}

func IsProjectNotFound(err error) bool {
	var target *ProjectNotFoundError
	return errors.As(err, &target)
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
