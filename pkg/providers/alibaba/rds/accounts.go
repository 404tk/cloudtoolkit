package rds

import (
	"context"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func (d *Driver) CreateAccount(instanceID, dbName string) bool {
	accountName, accountPassword, ok := parseRDSAccount()
	if !ok {
		return false
	}
	client := d.newClient()
	region := api.NormalizeRegion(d.Region)

	if _, err := client.CreateRDSAccount(context.Background(), region, instanceID, accountName, accountPassword); err != nil {
		logger.Error(err)
		return false
	}
	if err := grantAccountPrivilege(context.Background(), client, region, instanceID, accountName, dbName); err != nil {
		logger.Error(err)
		return false
	}
	fmt.Printf("\n%-10s\t%-10s\t%-60s\n", "Username", "Password", "Privilege")
	fmt.Printf("%-10s\t%-10s\t%-60s\n", "--------", "--------", "---------")
	fmt.Printf("%-10s\t%-10s\t%-60s\n\n",
		accountName, accountPassword, "ReadOnly")
	return true
}

func (d *Driver) DeleteAccount(instanceID string) {
	accountName, _, ok := parseRDSAccount()
	if !ok {
		return
	}
	client := d.newClient()
	region := api.NormalizeRegion(d.Region)

	if _, err := client.DeleteRDSAccount(context.Background(), region, instanceID, accountName); err != nil {
		logger.Error(err)
		return
	}
	logger.Warning(accountName + " user delete completed.")
}

func grantAccountPrivilege(ctx context.Context, client *api.Client, region, instanceID, userName, dbName string) error {
	_, err := client.GrantRDSAccountPrivilege(ctx, region, instanceID, userName, dbName, "ReadOnly")
	return err
}

func parseRDSAccount() (string, string, bool) {
	accountName, accountPassword, ok := strings.Cut(utils.RDSAccount, ":")
	if !ok || strings.TrimSpace(accountName) == "" || strings.TrimSpace(accountPassword) == "" {
		logger.Error("RDS account metadata is invalid. expected username:password")
		return "", "", false
	}
	return accountName, accountPassword, true
}
