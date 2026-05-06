package rds

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// CreateAccount provisions a Huawei RDS database account on the named
// instance. The endpoint family is `/v3/{project}/instances/{id}/db_user`
// (POST). engine routing is left to RDS itself — the MySQL path also serves
// PostgreSQL with the same payload shape.
func (d *Driver) CreateAccount(ctx context.Context, region, instanceID string) (schema.DatabaseActionResult, error) {
	user, password, err := parseRDSAccount()
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	region = strings.TrimSpace(region)
	if region == "" {
		region = d.requestRegion()
	}
	projectID, err := d.resolveProjectID(ctx, region)
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	body, err := json.Marshal(api.CreateRDSDBUserRequest{
		Name:     user,
		Password: password,
		Hosts:    []string{"%"},
		Comment:  "ctk validation",
	})
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	var resp api.CreateRDSDBUserResponse
	if err := d.client().DoJSON(ctx, api.Request{
		Service: "rds",
		Region:  region,
		Intl:    d.Cred.Intl,
		Method:  http.MethodPost,
		Path:    fmt.Sprintf("/v3/%s/instances/%s/db_user", url.PathEscape(projectID), url.PathEscape(instanceID)),
		Body:    body,
	}, &resp); err != nil {
		return schema.DatabaseActionResult{}, err
	}
	return schema.DatabaseActionResult{
		Action:   "useradd",
		Username: user,
		Password: password,
		Message:  fmt.Sprintf("RDS account created on %s", instanceID),
	}, nil
}

// DeleteAccount revokes the account named by the `rds-account-check` config.
func (d *Driver) DeleteAccount(ctx context.Context, region, instanceID string) (schema.DatabaseActionResult, error) {
	user, _, err := parseRDSAccount()
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	region = strings.TrimSpace(region)
	if region == "" {
		region = d.requestRegion()
	}
	projectID, err := d.resolveProjectID(ctx, region)
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	var resp api.DeleteRDSDBUserResponse
	if err := d.client().DoJSON(ctx, api.Request{
		Service: "rds",
		Region:  region,
		Intl:    d.Cred.Intl,
		Method:  http.MethodDelete,
		Path:    fmt.Sprintf("/v3/%s/instances/%s/db_user/%s", url.PathEscape(projectID), url.PathEscape(instanceID), url.PathEscape(user)),
	}, &resp); err != nil {
		return schema.DatabaseActionResult{}, err
	}
	return schema.DatabaseActionResult{
		Action:   "userdel",
		Username: user,
		Message:  user + " account delete completed.",
	}, nil
}

func (d *Driver) requestRegion() string {
	for _, r := range d.Regions {
		r = strings.TrimSpace(r)
		if r != "" && r != "all" {
			return r
		}
	}
	return ""
}

func parseRDSAccount() (string, string, error) {
	accountName, accountPassword, ok := strings.Cut(env.Active().RDSAccount, ":")
	if !ok || strings.TrimSpace(accountName) == "" || strings.TrimSpace(accountPassword) == "" {
		return "", "", fmt.Errorf("RDS account metadata is invalid: expected username:password")
	}
	return accountName, accountPassword, nil
}
