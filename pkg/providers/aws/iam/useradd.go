package iam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/aws/internal/arnutil"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const adminPolicyARN = "arn:aws:iam::aws:policy/AdministratorAccess"

func (d *Driver) AddUser() {
	ctx := context.Background()
	client, err := d.requireClient()
	if err != nil {
		logger.Error(err)
		return
	}
	region := d.requestRegion()

	accountArn, err := createUser(ctx, client, region, d.Username)
	if err != nil {
		logger.Error("Create user failed:", err)
		if !isEntityAlreadyExists(err) {
			return
		}
	}
	err = createLoginProfile(ctx, client, region, d.Username, d.Password)
	if err != nil {
		logger.Error("Create login password failed:", err)
		return
	}
	err = attachPolicyToUser(ctx, client, region, d.Username)
	if err != nil {
		logger.Error("Grant AdministratorAccess policy failed.")
		return
	}
	url := arnutil.ConsoleURLForARN(accountArn)
	fmt.Printf("\n%-10s\t%-20s\t%-60s\n", "Username", "Password", "Login URL")
	fmt.Printf("%-10s\t%-20s\t%-60s\n", "--------", "--------", "---------")
	fmt.Printf("%-10s\t%-20s\t%-60s\n\n", d.Username, d.Password, url)
}

func createUser(ctx context.Context, client *api.Client, region, userName string) (string, error) {
	resp, err := client.CreateUser(ctx, region, userName)
	if err != nil {
		return "", err
	}
	return resp.Arn, nil
}

func createLoginProfile(ctx context.Context, client *api.Client, region, userName, password string) error {
	return client.CreateLoginProfile(ctx, region, userName, password)
}

func attachPolicyToUser(ctx context.Context, client *api.Client, region, userName string) error {
	return client.AttachUserPolicy(ctx, region, userName, adminPolicyARN)
}

func isEntityAlreadyExists(err error) bool {
	return api.ErrorCode(err) == "EntityAlreadyExists"
}
