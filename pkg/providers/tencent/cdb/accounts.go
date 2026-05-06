package cdb

import (
	"context"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

const accountHost = "%"

// CreateAccount provisions a CDB account on the named instance using the
// `rds-account-check` config. The username/password come from
// `env.Active().RDSAccount` (form `username:password`).
func (d *Driver) CreateAccount(ctx context.Context, instanceID string) (schema.DatabaseActionResult, error) {
	user, password, err := parseRDSAccount()
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	region := normalizedRegion(d.Region)
	if _, err := d.newClient().CreateCDBAccounts(ctx, region, instanceID, user, accountHost, password); err != nil {
		return schema.DatabaseActionResult{}, err
	}
	return schema.DatabaseActionResult{
		Action:   "useradd",
		Username: user,
		Password: password,
		Message:  "CDB account created",
	}, nil
}

// DeleteAccount removes the CDB account named by the `rds-account-check`
// config from the supplied instance.
func (d *Driver) DeleteAccount(ctx context.Context, instanceID string) (schema.DatabaseActionResult, error) {
	user, _, err := parseRDSAccount()
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	region := normalizedRegion(d.Region)
	if _, err := d.newClient().DeleteCDBAccounts(ctx, region, instanceID, user, accountHost); err != nil {
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

// LookupInstance scans CDB to confirm an instanceID is reachable. Used by the
// shell helper to look up region context for region=all sessions.
func (d *Driver) LookupInstance(ctx context.Context, instanceID string) (api.CDBInstanceInfo, bool, error) {
	region := normalizedRegion(d.Region)
	resp, err := d.newClient().DescribeCDBInstances(ctx, region)
	if err != nil {
		return api.CDBInstanceInfo{}, false, err
	}
	for _, inst := range resp.Response.Items {
		if derefString(inst.InstanceID) == instanceID {
			return inst, true, nil
		}
	}
	return api.CDBInstanceInfo{}, false, nil
}
