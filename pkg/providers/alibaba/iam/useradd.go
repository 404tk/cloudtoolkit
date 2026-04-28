package iam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

func (d *Driver) AddUser() (schema.IAMResult, error) {
	ctx := context.Background()
	client := d.newClient()
	region := api.NormalizeRegion(d.Region)

	err := createUser(ctx, client, region, d.UserName)
	if err != nil {
		return schema.IAMResult{}, fmt.Errorf("create user failed: %w", err)
	}
	err = createLoginProfile(ctx, client, region, d.UserName, d.Password)
	if err != nil {
		return schema.IAMResult{}, fmt.Errorf("create login password failed: %w", err)
	}
	err = attachPolicyToUser(ctx, client, region, d.UserName)
	if err != nil {
		return schema.IAMResult{}, fmt.Errorf("grant AdministratorAccess policy failed: %w", err)
	}
	accountAlias := getAccountAlias(ctx, client, region)
	loginURL := fmt.Sprintf("https://signin.aliyun.com/%s/login.htm", accountAlias)

	return schema.IAMResult{
		Username:  d.UserName,
		Password:  d.Password,
		LoginURL:  loginURL,
		AccountID: accountAlias,
		Message:   "User created successfully with AdministratorAccess policy",
	}, nil
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
		return ""
	}
	return response.AccountAlias
}
