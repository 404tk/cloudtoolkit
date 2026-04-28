package iam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

func (d *Driver) AddUser() (schema.IAMResult, error) {
	ctx := context.Background()
	client := d.newClient()
	err := createUser(ctx, client, d.UserName, d.Password)
	if err != nil {
		return schema.IAMResult{}, fmt.Errorf("create user failed: %w", err)
	}
	_ = attachPolicyToUser(ctx, client, d.UserName)
	OwnerID := getOwnerUin(ctx, client)
	loginURL := "https://cloud.tencent.com/login/subAccount/" + OwnerID

	return schema.IAMResult{
		Username:  d.UserName,
		Password:  d.Password,
		LoginURL:  loginURL,
		AccountID: OwnerID,
		Message:   "User created successfully with AdministratorAccess policy",
	}, nil
}

func createUser(ctx context.Context, client *api.Client, userName, password string) error {
	_, err := client.AddUser(ctx, userName, password)
	return err
}

func attachPolicyToUser(ctx context.Context, client *api.Client, userName string) error {
	resp, err := getUserInfo(ctx, client, userName)
	if err != nil {
		return err
	}
	_, err = client.AttachUserPolicy(ctx, derefUint64(resp.Response.Uin), 1)
	return err
}

func getUserInfo(ctx context.Context, client *api.Client, userName string) (api.GetUserResponse, error) {
	return client.GetUser(ctx, userName)
}

func getOwnerUin(ctx context.Context, client *api.Client) string {
	response, err := client.GetUserAppID(ctx)
	if err != nil {
		return ""
	}
	return derefString(response.Response.OwnerUin)
}
