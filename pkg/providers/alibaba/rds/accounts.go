package rds

import (
	"context"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

func (d *Driver) CreateAccount(ctx context.Context, instanceID, dbName string) (schema.DatabaseActionResult, error) {
	accountName, accountPassword, err := parseRDSAccount()
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	client := d.newClient()
	region := api.NormalizeRegion(d.Region)

	if _, err := client.CreateRDSAccount(ctx, region, instanceID, accountName, accountPassword); err != nil {
		return schema.DatabaseActionResult{}, err
	}
	if err := grantAccountPrivilege(ctx, client, region, instanceID, accountName, dbName); err != nil {
		return schema.DatabaseActionResult{}, err
	}
	return schema.DatabaseActionResult{
		Action:    "useradd",
		Username:  accountName,
		Password:  accountPassword,
		Privilege: "ReadOnly",
		Message:   "database account created",
	}, nil
}

func (d *Driver) DeleteAccount(ctx context.Context, instanceID string) (schema.DatabaseActionResult, error) {
	accountName, _, err := parseRDSAccount()
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	client := d.newClient()
	region := api.NormalizeRegion(d.Region)

	if _, err := client.DeleteRDSAccount(ctx, region, instanceID, accountName); err != nil {
		return schema.DatabaseActionResult{}, err
	}
	return schema.DatabaseActionResult{
		Action:   "userdel",
		Username: accountName,
		Message:  accountName + " user delete completed.",
	}, nil
}

func grantAccountPrivilege(ctx context.Context, client *api.Client, region, instanceID, userName, dbName string) error {
	_, err := client.GrantRDSAccountPrivilege(ctx, region, instanceID, userName, dbName, "ReadOnly")
	return err
}

func parseRDSAccount() (string, string, error) {
	accountName, accountPassword, ok := strings.Cut(env.Active().RDSAccount, ":")
	if !ok || strings.TrimSpace(accountName) == "" || strings.TrimSpace(accountPassword) == "" {
		return "", "", fmt.Errorf("RDS account metadata is invalid: expected username:password")
	}
	return accountName, accountPassword, nil
}
