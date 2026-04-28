package iam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

func (d *Driver) DelRole() (schema.IAMResult, error) {
	ctx := context.Background()
	client := d.newClient()
	err := detachPolicyFromRole(ctx, client, d.RoleName)
	if err != nil {
		return schema.IAMResult{}, fmt.Errorf("remove policy from %s failed: %w", d.RoleName, err)
	}
	err = deleteRole(ctx, client, d.RoleName)
	if err != nil {
		return schema.IAMResult{}, fmt.Errorf("delete role %s failed: %w", d.RoleName, err)
	}

	return schema.IAMResult{
		Username: d.RoleName,
		Message:  "Role deleted successfully",
	}, nil
}

func detachPolicyFromRole(ctx context.Context, client *api.Client, roleName string) error {
	_, err := client.DetachRolePolicy(ctx, roleName, 1)
	return err
}

func deleteRole(ctx context.Context, client *api.Client, roleName string) error {
	_, err := client.DeleteRole(ctx, roleName)
	return err
}
