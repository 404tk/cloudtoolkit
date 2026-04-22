package iam

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const (
	administratorPolicyName = "AdministratorAccess"
	systemPolicyType        = "System"
	volcengineSigninURL     = "https://console.volcengine.com/auth/login/user/%s"
)

func (d *Driver) AddUser() {
	ctx := context.Background()
	client, err := d.requireClient()
	if err != nil {
		logger.Error(err)
		return
	}

	region := d.requestRegion()
	userName := strings.TrimSpace(d.UserName)
	password := d.Password
	if userName == "" {
		logger.Error("Empty user name.")
		return
	}
	if password == "" {
		logger.Error("Empty password.")
		return
	}

	if err := createUser(ctx, client, region, userName); err != nil {
		logger.Error("Create user failed:", err)
		return
	}
	if err := createLoginProfile(ctx, client, region, userName, password); err != nil {
		logger.Error("Create login profile failed:", err)
		return
	}
	if err := attachUserPolicy(ctx, client, region, userName); err != nil {
		logger.Error("Grant AdministratorAccess policy failed:", err)
		return
	}

	accountID := currentAccountID(ctx, client, region)
	fmt.Printf("\n%-16s\t%-16s\t%-40s\n", "Username", "Password", "Login URL")
	fmt.Printf("%-16s\t%-16s\t%-40s\n", "--------", "--------", "---------")
	fmt.Printf("%-16s\t%-16s\t%-40s\n", userName, password,
		fmt.Sprintf(volcengineSigninURL, accountID))
	fmt.Println()
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
		logger.Error("Get account ID failed:", err)
		return ""
	}
	if len(resp.Result.Projects) == 0 {
		return ""
	}
	return strconv.FormatInt(resp.Result.Projects[0].AccountID, 10)
}
