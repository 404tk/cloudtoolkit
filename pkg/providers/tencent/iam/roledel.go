package iam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func (d *Driver) DelRole() {
	ctx := context.Background()
	client := d.newClient()
	err := detachPolicyFromRole(ctx, client, d.RoleName)
	if err != nil {
		logger.Error(fmt.Sprintf("Remove policy from %s failed: %s", d.RoleName, err.Error()))
		return
	}
	err = deleteRole(ctx, client, d.RoleName)
	if err != nil {
		logger.Error(fmt.Sprintf("Delete role %s failed: %s", d.RoleName, err.Error()))
		return
	}
	logger.Warning(d.RoleName + " role delete completed.")
}

func detachPolicyFromRole(ctx context.Context, client *api.Client, roleName string) error {
	_, err := client.DetachRolePolicy(ctx, roleName, 1)
	return err
}

func deleteRole(ctx context.Context, client *api.Client, roleName string) error {
	_, err := client.DeleteRole(ctx, roleName)
	return err
}
