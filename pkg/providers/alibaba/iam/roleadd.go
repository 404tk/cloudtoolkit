package iam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func (d *Driver) AddRole() {
	ctx := context.Background()
	client := d.newClient()
	region := api.NormalizeRegion(d.Region)

	err := createRole(ctx, client, region, d.RoleName, d.AccountId)
	if err != nil {
		logger.Error("Create role failed:", err.Error())
		return
	}
	err = attachPolicyToRole(ctx, client, region, d.RoleName)
	if err != nil {
		logger.Error("Grant AdministratorAccess policy failed.")
		return
	}
	accountAlias := getAccountAlias(ctx, client, region)
	fmt.Printf("\n%-20s\t%-10s\t%-60s\n", "AccountAlias", "RoleName", "Switch URL")
	fmt.Printf("%-20s\t%-10s\t%-60s\n", "------------", "--------", "----------")
	fmt.Printf("%-20s\t%-10s\t%-60s\n\n",
		accountAlias, d.RoleName,
		"https://signin.aliyun.com/switchRole.htm")
}

func createRole(ctx context.Context, client *api.Client, region, roleName, accountId string) error {
	assumeRolePolicyDocument := fmt.Sprintf(
		"{\"Statement\":[{\"Action\":\"sts:AssumeRole\",\"Effect\":\"Allow\",\"Principal\":{\"RAM\":\"acs:ram::%s:root\"}}],\"Version\":\"1\"}",
		accountId)
	_, err := client.CreateRAMRole(ctx, region, roleName, assumeRolePolicyDocument)
	return err
}

func attachPolicyToRole(ctx context.Context, client *api.Client, region, roleName string) error {
	_, err := client.AttachRAMPolicyToRole(ctx, region, roleName, "AdministratorAccess", "System")
	return err
}
