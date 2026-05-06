package udb

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

func (d *Driver) requestRegion() string {
	for _, region := range d.Regions {
		region = strings.TrimSpace(region)
		if region != "" && region != "all" {
			return region
		}
	}
	return ""
}

// CreateAccount provisions a UDB account on the named instance using the
// `rds-account-check` config. UDB accounts have no host concept — the API
// requires UserName/Password and a Permission flag (`Normal`).
func (d *Driver) CreateAccount(ctx context.Context, instanceID string) (schema.DatabaseActionResult, error) {
	if d == nil {
		return schema.DatabaseActionResult{}, errors.New("ucloud udb: nil driver")
	}
	user, password, err := parseRDSAccount()
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	region := d.requestRegion()
	if region == "" {
		return schema.DatabaseActionResult{}, errors.New("ucloud udb: empty region")
	}
	params := map[string]any{
		"Region":     region,
		"DBId":       instanceID,
		"UserName":   user,
		"Password":   password,
		"Permission": "Normal",
	}
	if d.ProjectID != "" {
		params["ProjectId"] = d.ProjectID
	}
	var resp api.CreateUDBUserResponse
	if err := d.client().Do(ctx, api.Request{Action: "CreateUDBUser", Params: params}, &resp); err != nil {
		return schema.DatabaseActionResult{}, err
	}
	return schema.DatabaseActionResult{
		Action:   "useradd",
		Username: user,
		Password: password,
		Message:  fmt.Sprintf("UDB account created on %s", instanceID),
	}, nil
}

// DeleteAccount removes the UDB account named by `rds-account-check` from
// instanceID.
func (d *Driver) DeleteAccount(ctx context.Context, instanceID string) (schema.DatabaseActionResult, error) {
	if d == nil {
		return schema.DatabaseActionResult{}, errors.New("ucloud udb: nil driver")
	}
	user, _, err := parseRDSAccount()
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	region := d.requestRegion()
	if region == "" {
		return schema.DatabaseActionResult{}, errors.New("ucloud udb: empty region")
	}
	params := map[string]any{
		"Region":   region,
		"DBId":     instanceID,
		"UserName": user,
	}
	if d.ProjectID != "" {
		params["ProjectId"] = d.ProjectID
	}
	var resp api.DeleteUDBUserResponse
	if err := d.client().Do(ctx, api.Request{Action: "DeleteUDBUser", Params: params}, &resp); err != nil {
		return schema.DatabaseActionResult{}, err
	}
	return schema.DatabaseActionResult{
		Action:   "userdel",
		Username: user,
		Message:  user + " account delete completed.",
	}, nil
}

func parseRDSAccount() (string, string, error) {
	accountName, accountPassword, ok := strings.Cut(env.Active().RDSAccount, ":")
	if !ok || strings.TrimSpace(accountName) == "" || strings.TrimSpace(accountPassword) == "" {
		return "", "", fmt.Errorf("RDS account metadata is invalid: expected username:password")
	}
	return accountName, accountPassword, nil
}
