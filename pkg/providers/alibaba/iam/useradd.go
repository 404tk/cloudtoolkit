package iam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func (d *Driver) AddUser() {
	ctx := context.Background()
	client := d.newClient()
	region := api.NormalizeRegion(d.Region)

	err := createUser(ctx, client, region, d.UserName)
	if err != nil {
		logger.Error("Create user failed:", err.Error())
		return
	}
	err = createLoginProfile(ctx, client, region, d.UserName, d.Password)
	if err != nil {
		logger.Error("Create login password failed:", err.Error())
		return
	}
	err = attachPolicyToUser(ctx, client, region, d.UserName)
	if err != nil {
		logger.Error("Grant AdministratorAccess policy failed.")
		return
	}
	accountAlias := getAccountAlias(ctx, client, region)
	fmt.Printf("\n%-10s\t%-10s\t%-60s\n", "Username", "Password", "Login URL")
	fmt.Printf("%-10s\t%-10s\t%-60s\n", "--------", "--------", "---------")
	fmt.Printf("%-10s\t%-10s\t%-60s\n\n",
		d.UserName, d.Password,
		fmt.Sprintf("https://signin.aliyun.com/%s/login.htm", accountAlias))
}

func createUser(ctx context.Context, client *api.Client, region, userName string) error {
	_, err := client.CreateRAMUser(ctx, region, userName)
	return err
}

func createLoginProfile(ctx context.Context, client *api.Client, region, userName, password string) error {
	_, err := client.CreateRAMLoginProfile(ctx, region, userName, password)
	return err
}

func attachPolicyToUser(ctx context.Context, client *api.Client, region, userName string) error {
	_, err := client.AttachRAMPolicyToUser(ctx, region, userName, "AdministratorAccess", "System")
	return err
}

func getAccountAlias(ctx context.Context, client *api.Client, region string) string {
	response, err := client.GetRAMAccountAlias(ctx, region)
	if err != nil {
		logger.Error("Get account alias failed.")
		return ""
	}
	return response.AccountAlias
}
