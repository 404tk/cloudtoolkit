package iam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func (d *Driver) AddRole() {
	ctx := context.Background()
	client := d.newClient()
	err := createRole(ctx, client, d.RoleName, d.Uin)
	if err != nil {
		logger.Error("Create role failed:", err.Error())
		return
	}
	_ = attachPolicyToRole(ctx, client, d.RoleName)
	OwnerID := getOwnerUin(ctx, client)
	logger.Warning(fmt.Sprintf(
		"Switch URL: https://cloud.tencent.com/cam/switchrole?ownerUin=%s&roleName=%s\n",
		OwnerID, d.RoleName))
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
