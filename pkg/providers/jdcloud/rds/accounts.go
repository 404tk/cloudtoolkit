// Package rds wraps JDCloud RDS account lifecycle. Endpoint paths are
// pattern-inferred from JDCloud's REST convention; verify against the
// upstream SDK before relying on this in production.
package rds

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

type Driver struct {
	Client *api.Client
	Region string
}

// CreateAccount provisions an RDS account on instanceID via the standard
// `/v1/regions/<region>/instances/<id>/accounts` POST.
func (d *Driver) CreateAccount(ctx context.Context, instanceID string) (schema.DatabaseActionResult, error) {
	if d == nil || d.Client == nil {
		return schema.DatabaseActionResult{}, errors.New("jdcloud rds: nil api client")
	}
	user, password, err := parseRDSAccount()
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	body, err := json.Marshal(api.CreateRDSAccountRequest{AccountName: user, AccountPassword: password})
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	if _, err := d.Client.CreateRDSAccount(ctx, d.Region, instanceID, body); err != nil {
		return schema.DatabaseActionResult{}, err
	}
	return schema.DatabaseActionResult{
		Action:   "useradd",
		Username: user,
		Password: password,
		Message:  fmt.Sprintf("RDS account created on %s", instanceID),
	}, nil
}

// DeleteAccount revokes the RDS account named by `rds-account-check` from
// instanceID.
func (d *Driver) DeleteAccount(ctx context.Context, instanceID string) (schema.DatabaseActionResult, error) {
	if d == nil || d.Client == nil {
		return schema.DatabaseActionResult{}, errors.New("jdcloud rds: nil api client")
	}
	user, _, err := parseRDSAccount()
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	if _, err := d.Client.DeleteRDSAccount(ctx, d.Region, instanceID, user); err != nil {
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
