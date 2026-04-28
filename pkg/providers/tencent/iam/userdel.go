package iam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

func (d *Driver) DelUser() (schema.IAMResult, error) {
	ctx := context.Background()
	client := d.newClient()
	err := detachPolicyFromUser(ctx, client, d.UserName)
	if err != nil {
		return schema.IAMResult{}, fmt.Errorf("remove policy from %s failed: %w", d.UserName, err)
	}
	err = deleteUser(ctx, client, d.UserName)
	if err != nil {
		return schema.IAMResult{}, fmt.Errorf("delete user %s failed: %w", d.UserName, err)
	}

	return schema.IAMResult{
		Username: d.UserName,
		Message:  "User deleted successfully",
	}, nil
}

func detachPolicyFromUser(ctx context.Context, client *api.Client, userName string) error {
	resp, err := getUserInfo(ctx, client, userName)
	if err != nil {
		return err
	}
	_, err = client.DetachUserPolicy(ctx, derefUint64(resp.Response.Uin), 1)
	return err
}

func deleteUser(ctx context.Context, client *api.Client, userName string) error {
	_, err := client.DeleteUser(ctx, userName, 1)
	return err
}
