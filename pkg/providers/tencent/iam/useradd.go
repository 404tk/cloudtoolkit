package iam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func (d *Driver) AddUser() {
	ctx := context.Background()
	client := d.newClient()
	err := createUser(ctx, client, d.UserName, d.Password)
	if err != nil {
		logger.Error("Create user failed:", err.Error())
		return
	}
	_ = attachPolicyToUser(ctx, client, d.UserName)
	OwnerID := getOwnerUin(ctx, client)
	fmt.Printf("\n%-10s\t%-10s\t%-60s\n", "Username", "Password", "Login URL")
	fmt.Printf("%-10s\t%-10s\t%-60s\n", "--------", "--------", "---------")
	fmt.Printf("%-10s\t%-10s\t%-60s\n\n",
		d.UserName,
		d.Password, "https://cloud.tencent.com/login/subAccount/"+OwnerID)
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
		logger.Error("Get user appid failed.")
		return ""
	}
	return derefString(response.Response.OwnerUin)
}
