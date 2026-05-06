// Package rds wraps AWS RDS master password rotation. AWS RDS doesn't expose
// per-user create/delete via API — accounts live in the database engine. The
// closest CSPM-detectable management-plane signal is `ModifyDBInstance` with
// MasterUserPassword, captured by CloudTrail.
package rds

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

type Driver struct {
	Client        *api.Client
	Region        string
	DefaultRegion string
}

// CreateAccount rotates the RDS master password to the value supplied by the
// `rds-account-check` config — equivalent to "set a known password on the
// master user". The username comes from the existing instance.
func (d *Driver) CreateAccount(ctx context.Context, instanceID string) (schema.DatabaseActionResult, error) {
	if d == nil || d.Client == nil {
		return schema.DatabaseActionResult{}, errors.New("aws rds: nil api client")
	}
	configUser, password, err := parseRDSAccount()
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	region := d.requestRegion()
	if region == "" {
		return schema.DatabaseActionResult{}, errors.New("aws rds: explicit region required")
	}
	out, err := d.Client.ModifyDBInstanceMasterPassword(ctx, region, instanceID, password)
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	displayUser := out.MasterUsername
	if displayUser == "" {
		displayUser = configUser
	}
	return schema.DatabaseActionResult{
		Action:    "useradd",
		Username:  displayUser,
		Password:  password,
		Privilege: "MasterUser",
		Message:   fmt.Sprintf("RDS master password rotated to known value on %s (instance status %s)", out.DBInstanceIdentifier, out.DBInstanceStatus),
	}, nil
}

// DeleteAccount rotates the RDS master password to a fresh random value to
// revoke the access granted by `useradd`.
func (d *Driver) DeleteAccount(ctx context.Context, instanceID string) (schema.DatabaseActionResult, error) {
	if d == nil || d.Client == nil {
		return schema.DatabaseActionResult{}, errors.New("aws rds: nil api client")
	}
	region := d.requestRegion()
	if region == "" {
		return schema.DatabaseActionResult{}, errors.New("aws rds: explicit region required")
	}
	password, err := api.RandomPassword()
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	out, err := d.Client.ModifyDBInstanceMasterPassword(ctx, region, instanceID, password)
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	return schema.DatabaseActionResult{
		Action:   "userdel",
		Username: out.MasterUsername,
		Message:  fmt.Sprintf("RDS master password rotated to random value on %s; access via known credential revoked", out.DBInstanceIdentifier),
	}, nil
}

func (d *Driver) requestRegion() string {
	region := strings.TrimSpace(d.Region)
	if region == "" || region == "all" {
		region = strings.TrimSpace(d.DefaultRegion)
	}
	return region
}

func parseRDSAccount() (string, string, error) {
	accountName, accountPassword, ok := strings.Cut(env.Active().RDSAccount, ":")
	if !ok || strings.TrimSpace(accountName) == "" || strings.TrimSpace(accountPassword) == "" {
		return "", "", fmt.Errorf("RDS account metadata is invalid: expected username:password")
	}
	return accountName, accountPassword, nil
}
