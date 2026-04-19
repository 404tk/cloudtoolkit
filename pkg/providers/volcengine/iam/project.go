package iam

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
)

var errNilAPIClient = errors.New("volcengine iam: nil api client")

func (d *Driver) GetProject(ctx context.Context) (string, error) {
	client, err := d.requireClient()
	if err != nil {
		return "", err
	}
	resp, err := client.ListProjects(ctx, d.requestRegion())
	if err != nil {
		return "", err
	}
	if len(resp.Result.Projects) == 0 {
		return "", errors.New("volcengine iam: empty project list")
	}
	project := resp.Result.Projects[0]
	return fmt.Sprintf("%s(%d)", strings.TrimSpace(project.ProjectName), project.AccountID), nil
}

func (d *Driver) requireClient() (*api.Client, error) {
	if d.Client == nil {
		return nil, errNilAPIClient
	}
	return d.Client, nil
}

func (d *Driver) requestRegion() string {
	region := strings.TrimSpace(d.Region)
	if region == "" || region == "all" {
		return api.DefaultRegion
	}
	return region
}
