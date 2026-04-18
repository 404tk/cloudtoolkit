package iam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func (d *Driver) DelUser() {
	ctx := context.Background()
	client, err := d.requireClient()
	if err != nil {
		logger.Error(err)
		return
	}
	region := d.requestRegion()

	err = deleteLoginProfile(ctx, client, region, d.Username)
	if err != nil {
		if !isNoSuchEntity(err) {
			logger.Error(fmt.Sprintf("Delete login profile failed: %s", err))
			return
		}
	}
	err = detachUserPolicy(ctx, client, region, d.Username)
	if err != nil {
		if !isNoSuchEntity(err) {
			logger.Error(fmt.Sprintf("Remove policy from %s failed: %s", d.Username, err))
			return
		}
	}
	err = deleteUser(ctx, client, region, d.Username)
	if err != nil {
		logger.Error(fmt.Sprintf("Delete user failed: %s", err))
		return
	}
	logger.Warning(fmt.Sprintf("Delete user %s success!", d.Username))
}

func detachUserPolicy(ctx context.Context, client *api.Client, region, userName string) error {
	return client.DetachUserPolicy(ctx, region, userName, adminPolicyARN)
}

func deleteLoginProfile(ctx context.Context, client *api.Client, region, userName string) error {
	return client.DeleteLoginProfile(ctx, region, userName)
}

func deleteUser(ctx context.Context, client *api.Client, region, userName string) error {
	return client.DeleteUser(ctx, region, userName)
}

func isNoSuchEntity(err error) bool {
	return api.ErrorCode(err) == "NoSuchEntity"
}
