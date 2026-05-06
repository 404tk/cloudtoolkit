package rds

import (
	"context"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// CreateAccount provisions a Volcengine RDS account on the named instance.
// Service (`rds_mysql` / `rds_postgresql` / `rds_mssql`) is auto-detected by
// the `instanceID` prefix when not provided. Username/password come from the
// `rds-account-check` config.
func (d *Driver) CreateAccount(ctx context.Context, instanceID string) (schema.DatabaseActionResult, error) {
	if d == nil || d.Client == nil {
		return schema.DatabaseActionResult{}, errNilAPIClient
	}
	user, password, err := parseRDSAccount()
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	service := serviceForInstance(instanceID)
	region := d.requestRegion()
	if _, err := d.Client.CreateRDSDBAccount(ctx, service, region, instanceID, user, password); err != nil {
		return schema.DatabaseActionResult{}, err
	}
	return schema.DatabaseActionResult{
		Action:   "useradd",
		Username: user,
		Password: password,
		Message:  fmt.Sprintf("RDS account created on %s", instanceID),
	}, nil
}

// DeleteAccount removes the RDS account named by `rds-account-check`.
func (d *Driver) DeleteAccount(ctx context.Context, instanceID string) (schema.DatabaseActionResult, error) {
	if d == nil || d.Client == nil {
		return schema.DatabaseActionResult{}, errNilAPIClient
	}
	user, _, err := parseRDSAccount()
	if err != nil {
		return schema.DatabaseActionResult{}, err
	}
	service := serviceForInstance(instanceID)
	region := d.requestRegion()
	if _, err := d.Client.DeleteRDSDBAccount(ctx, service, region, instanceID, user); err != nil {
		return schema.DatabaseActionResult{}, err
	}
	return schema.DatabaseActionResult{
		Action:   "userdel",
		Username: user,
		Message:  user + " account delete completed.",
	}, nil
}

// serviceForInstance maps an instance ID prefix to the Volcengine RDS sub-
// service. mysql / postgres / mssql instance IDs follow distinct prefixes;
// fall back to MySQL when the prefix is unknown.
func serviceForInstance(instanceID string) string {
	id := strings.ToLower(strings.TrimSpace(instanceID))
	switch {
	case strings.HasPrefix(id, "postgres-"), strings.HasPrefix(id, "pg-"), strings.HasPrefix(id, "rdspg-"):
		return api.ServiceRDSPostgreSQL
	case strings.HasPrefix(id, "mssql-"), strings.HasPrefix(id, "rdsmssql-"), strings.HasPrefix(id, "sqlserver-"):
		return api.ServiceRDSMSSQL
	default:
		return api.ServiceRDSMySQL
	}
}

func parseRDSAccount() (string, string, error) {
	accountName, accountPassword, ok := strings.Cut(env.Active().RDSAccount, ":")
	if !ok || strings.TrimSpace(accountName) == "" || strings.TrimSpace(accountPassword) == "" {
		return "", "", fmt.Errorf("RDS account metadata is invalid: expected username:password")
	}
	return accountName, accountPassword, nil
}
