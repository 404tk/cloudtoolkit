package iam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func (d *Driver) AddRole() (schema.IAMResult, error) {
	ctx := context.Background()
	client := d.newClient()
	err := createRole(ctx, client, d.RoleName, d.Uin)
	if err != nil {
		return schema.IAMResult{}, fmt.Errorf("create role failed: %w", err)
	}
	message := "Role created successfully with AdministratorAccess policy"
	if err := attachPolicyToRole(ctx, client, d.RoleName); err != nil {
		logger.Warning("attach AdministratorAccess policy to role failed:", err)
		message = "Role created, but failed to attach AdministratorAccess policy: " + err.Error()
	}
	ownerID := getOwnerUin(ctx, client)
	switchURL := fmt.Sprintf("https://cloud.tencent.com/cam/switchrole?ownerUin=%s&roleName=%s", ownerID, d.RoleName)

	return schema.IAMResult{
		Username:  d.RoleName,
		AccountID: ownerID,
		LoginURL:  switchURL,
		Message:   message,
	}, nil
}

func createRole(ctx context.Context, client *api.Client, roleName, uin string) error {
	policy := fmt.Sprintf(
		`{"version":"2.0","statement":[{"action":"name/sts:AssumeRole","effect":"allow","principal":{"qcs":["qcs::cam::uin/%s:root"]}}]}`, uin)
	_, err := client.CreateRole(ctx, roleName, policy, 1, 10000)
	return err
}

func attachPolicyToRole(ctx context.Context, client *api.Client, roleName string) error {
	_, err := client.AttachRolePolicy(ctx, roleName, 1)
	return err
}
