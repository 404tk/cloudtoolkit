package iam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/aws/internal/arnutil"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

const adminPolicyARN = "arn:aws:iam::aws:policy/AdministratorAccess"

func (d *Driver) AddUser() (schema.IAMResult, error) {
	ctx := context.Background()
	client, err := d.requireClient()
	if err != nil {
		return schema.IAMResult{}, err
	}
	region := d.requestRegion()

	accountArn, err := createUser(ctx, client, region, d.Username)
	if err != nil {
		if !isEntityAlreadyExists(err) {
			return schema.IAMResult{}, fmt.Errorf("create user failed: %w", err)
		}
	}
	err = createLoginProfile(ctx, client, region, d.Username, d.Password)
	if err != nil {
		return schema.IAMResult{}, fmt.Errorf("create login password failed: %w", err)
	}
	err = attachPolicyToUser(ctx, client, region, d.Username)
	if err != nil {
		return schema.IAMResult{}, fmt.Errorf("grant AdministratorAccess policy failed: %w", err)
	}
	url := arnutil.ConsoleURLForARN(accountArn)

	return schema.IAMResult{
		Username: d.Username,
		Password: d.Password,
		LoginURL: url,
		Message:  "User created successfully with AdministratorAccess policy",
	}, nil
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
