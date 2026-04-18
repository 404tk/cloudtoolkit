package iam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func (d *Driver) DelUser() {
	ctx := context.Background()
	client := d.newClient()
	err := detachPolicyFromUser(ctx, client, d.UserName)
	if err != nil {
		logger.Error(fmt.Sprintf("Remove policy from %s failed: %s", d.UserName, err.Error()))
		return
	}
	err = deleteUser(ctx, client, d.UserName)
	if err != nil {
		logger.Error(fmt.Sprintf("Delete user %s failed: %s", d.UserName, err.Error()))
		return
	}
	logger.Warning(d.UserName + " user delete completed.")
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
