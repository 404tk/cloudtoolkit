package iam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

func (d *Driver) AddRole() (schema.IAMResult, error) {
	ctx := context.Background()
	client := d.newClient()
	region := api.NormalizeRegion(d.Region)

	err := createRole(ctx, client, region, d.RoleName, d.AccountId)
	if err != nil {
		return schema.IAMResult{}, fmt.Errorf("create role failed: %w", err)
	}
	err = attachPolicyToRole(ctx, client, region, d.RoleName)
	if err != nil {
		return schema.IAMResult{}, fmt.Errorf("grant AdministratorAccess policy failed: %w", err)
	}
	accountAlias := getAccountAlias(ctx, client, region)

	return schema.IAMResult{
		Username:  d.RoleName,
		AccountID: accountAlias,
		LoginURL:  "https://signin.aliyun.com/switchRole.htm",
		Message:   "Role created successfully with AdministratorAccess policy",
	}, nil
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
