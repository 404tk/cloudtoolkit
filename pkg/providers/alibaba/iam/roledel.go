package iam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func (d *Driver) DelRole() {
	ctx := context.Background()
	client := d.newClient()
	region := api.NormalizeRegion(d.Region)

	err := detachPolicyFromRole(ctx, client, region, d.RoleName)
	if err != nil {
		logger.Error(fmt.Sprintf("Remove policy from %s failed: %s", d.RoleName, err.Error()))
		return
	}
	err = deleteRole(ctx, client, region, d.RoleName)
	if err != nil {
		logger.Error(fmt.Sprintf("Delete role %s failed: %s", d.RoleName, err.Error()))
		return
	}
	logger.Warning(d.RoleName + " role delete completed.")
}

func detachPolicyFromRole(ctx context.Context, client *api.Client, region, roleName string) error {
	_, err := client.DetachRAMPolicyFromRole(ctx, region, roleName, "AdministratorAccess", "System")
	return err
}

func deleteRole(ctx context.Context, client *api.Client, region, roleName string) error {
	_, err := client.DeleteRAMRole(ctx, region, roleName)
	return err
}
