package iam

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

const (
	administratorPolicyName = "AdministratorAccess"
	systemPolicyType        = "System"
	volcengineSigninURL     = "https://console.volcengine.com/auth/login/user/%s"
)

func (d *Driver) AddUser() (schema.IAMResult, error) {
	ctx := context.Background()
	client, err := d.requireClient()
	if err != nil {
		return schema.IAMResult{}, fmt.Errorf("require client failed: %w", err)
	}

	region := d.requestRegion()
	userName := strings.TrimSpace(d.UserName)
	password := d.Password
	if userName == "" {
		return schema.IAMResult{}, fmt.Errorf("empty user name")
	}
	if password == "" {
		return schema.IAMResult{}, fmt.Errorf("empty password")
	}

	if err := createUser(ctx, client, region, userName); err != nil {
		return schema.IAMResult{}, fmt.Errorf("create user failed: %w", err)
	}
	if err := createLoginProfile(ctx, client, region, userName, password); err != nil {
		return schema.IAMResult{}, fmt.Errorf("create login profile failed: %w", err)
	}
	if err := attachUserPolicy(ctx, client, region, userName); err != nil {
		return schema.IAMResult{}, fmt.Errorf("grant AdministratorAccess policy failed: %w", err)
	}

	accountID := currentAccountID(ctx, client, region)
	loginURL := fmt.Sprintf(volcengineSigninURL, accountID)

	return schema.IAMResult{
		Username:  userName,
		Password:  password,
		LoginURL:  loginURL,
		AccountID: accountID,
		Message:   "User created successfully with AdministratorAccess policy",
	}, nil
}

func createUser(ctx context.Context, client *api.Client, region, userName string) error {
	_, err := client.CreateUser(ctx, region, userName, userName)
	return err
}

func createLoginProfile(ctx context.Context, client *api.Client, region, userName, password string) error {
	_, err := client.CreateLoginProfile(ctx, region, userName, password)
	return err
}

func attachUserPolicy(ctx context.Context, client *api.Client, region, userName string) error {
	_, err := client.AttachUserPolicy(ctx, region, userName, administratorPolicyName, systemPolicyType)
	return err
}

func currentAccountID(ctx context.Context, client *api.Client, region string) string {
	resp, err := client.ListProjects(ctx, region)
	if err != nil {
		return ""
	}
	if len(resp.Result.Projects) == 0 {
		return ""
	}
	return strconv.FormatInt(resp.Result.Projects[0].AccountID, 10)
}
