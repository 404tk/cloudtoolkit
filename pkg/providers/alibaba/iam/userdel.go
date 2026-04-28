package iam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

func (d *Driver) DelUser() (schema.IAMResult, error) {
	ctx := context.Background()
	client := d.newClient()
	region := api.NormalizeRegion(d.Region)

	err := detachPolicyFromUser(ctx, client, region, d.UserName)
	if err != nil {
		if !isEntityNotExistError(err) {
			return schema.IAMResult{}, fmt.Errorf("remove policy from %s failed: %w", d.UserName, err)
		}
	}
	err = deleteUser(ctx, client, region, d.UserName)
	if err != nil {
		return schema.IAMResult{}, fmt.Errorf("delete user %s failed: %w", d.UserName, err)
	}

	return schema.IAMResult{
		Username: d.UserName,
		Message:  "User deleted successfully",
	}, nil
}

func detachPolicyFromUser(ctx context.Context, client *api.Client, region, userName string) error {
	_, err := client.DetachRAMPolicyFromUser(ctx, region, userName, "AdministratorAccess", "System")
	return err
}

func deleteUser(ctx context.Context, client *api.Client, region, userName string) error {
	_, err := client.DeleteRAMUser(ctx, region, userName)
	return err
}
