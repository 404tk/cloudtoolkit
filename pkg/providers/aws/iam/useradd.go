package iam

import (
	"context"
	"errors"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/internal/arnutil"
	"github.com/404tk/cloudtoolkit/utils/logger"
	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
)

func (d *Driver) AddUser() {
	ctx := context.Background()
	client := iam.NewFromConfig(d.Config)
	accountArn, err := createUser(ctx, client, d.Username)
	if err != nil {
		logger.Error("Create user failed:", err)
		if !isEntityAlreadyExists(err) {
			return
		}
	}
	err = createLoginProfile(ctx, client, d.Username, d.Password)
	if err != nil {
		logger.Error("Create login password failed:", err)
		return
	}
	err = attachPolicyToUser(ctx, client, d.Username)
	if err != nil {
		logger.Error("Grant AdministratorAccess policy failed.")
		return
	}
	url := arnutil.ConsoleURLForARN(accountArn)
	fmt.Printf("\n%-10s\t%-20s\t%-60s\n", "Username", "Password", "Login URL")
	fmt.Printf("%-10s\t%-20s\t%-60s\n", "--------", "--------", "---------")
	fmt.Printf("%-10s\t%-20s\t%-60s\n\n", d.Username, d.Password, url)
}

func createUser(ctx context.Context, client *iam.Client, userName string) (string, error) {
	resp, err := client.CreateUser(ctx, &iam.CreateUserInput{UserName: &userName})
	if err != nil {
		return "", err
	}
	return awsv2.ToString(resp.User.Arn), err
}

func createLoginProfile(ctx context.Context, client *iam.Client, userName string, password string) error {
	request := &iam.CreateLoginProfileInput{}
	request.UserName = &userName
	request.Password = &password
	_, err := client.CreateLoginProfile(ctx, request)
	return err
}

func attachPolicyToUser(ctx context.Context, client *iam.Client, userName string) error {
	request := &iam.AttachUserPolicyInput{}
	policyArn := "arn:aws:iam::aws:policy/AdministratorAccess"
	request.PolicyArn = &policyArn
	request.UserName = &userName
	_, err := client.AttachUserPolicy(ctx, request)
	return err
}

func isEntityAlreadyExists(err error) bool {
	var target *iamtypes.EntityAlreadyExistsException
	return errors.As(err, &target)
}
