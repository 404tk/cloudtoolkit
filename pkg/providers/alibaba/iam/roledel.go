package iam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

func (d *Driver) DelRole() (schema.IAMResult, error) {
	ctx := context.Background()
	client := d.newClient()
	region := api.NormalizeRegion(d.Region)

	err := detachPolicyFromRole(ctx, client, region, d.RoleName)
	if err != nil {
		return schema.IAMResult{}, fmt.Errorf("remove policy from %s failed: %w", d.RoleName, err)
	}
	err = deleteRole(ctx, client, region, d.RoleName)
	if err != nil {
		return schema.IAMResult{}, fmt.Errorf("delete role %s failed: %w", d.RoleName, err)
	}

	return schema.IAMResult{
		Username: d.RoleName,
		Message:  "Role deleted successfully",
	}, nil
}

func detachPolicyFromRole(ctx context.Context, client *api.Client, region, roleName string) error {
	_, err := client.DetachRAMPolicyFromRole(ctx, region, roleName, "AdministratorAccess", "System")
	return err
}

func deleteRole(ctx context.Context, client *api.Client, region, roleName string) error {
	_, err := client.DeleteRAMRole(ctx, region, roleName)
	return err
}
