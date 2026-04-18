package iam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func (d *Driver) DelUser() {
	ctx := context.Background()
	client := d.newClient()
	region := api.NormalizeRegion(d.Region)

	err := detachPolicyFromUser(ctx, client, region, d.UserName)
	if err != nil {
		if !isEntityNotExistError(err) {
			logger.Error(fmt.Sprintf("Remove policy from %s failed: %s", d.UserName, err))
			return
		}
	}
	err = deleteUser(ctx, client, region, d.UserName)
	if err != nil {
		logger.Error(fmt.Sprintf("Delete user %s failed: %s", d.UserName, err))
		return
	}
	logger.Warning(d.UserName + " user delete completed.")
}

func detachPolicyFromUser(ctx context.Context, client *api.Client, region, userName string) error {
	_, err := client.DetachRAMPolicyFromUser(ctx, region, userName, "AdministratorAccess", "System")
	return err
}

func deleteUser(ctx context.Context, client *api.Client, region, userName string) error {
	_, err := client.DeleteRAMUser(ctx, region, userName)
	return err
}
