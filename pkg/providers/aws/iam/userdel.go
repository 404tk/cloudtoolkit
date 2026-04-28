package iam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

func (d *Driver) DelUser() (schema.IAMResult, error) {
	ctx := context.Background()
	client, err := d.requireClient()
	if err != nil {
		return schema.IAMResult{}, err
	}
	region := d.requestRegion()

	err = deleteLoginProfile(ctx, client, region, d.Username)
	if err != nil {
		if !isNoSuchEntity(err) {
			return schema.IAMResult{}, fmt.Errorf("delete login profile failed: %w", err)
		}
	}
	err = detachUserPolicy(ctx, client, region, d.Username)
	if err != nil {
		if !isNoSuchEntity(err) {
			return schema.IAMResult{}, fmt.Errorf("remove policy from %s failed: %w", d.Username, err)
		}
	}
	err = deleteUser(ctx, client, region, d.Username)
	if err != nil {
		return schema.IAMResult{}, fmt.Errorf("delete user failed: %w", err)
	}

	return schema.IAMResult{
		Username: d.Username,
		Message:  "User deleted successfully",
	}, nil
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
