package iam

import (
	"context"
	"errors"
	"fmt"

	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
)

func (d *Driver) DelUser() {
	ctx := context.Background()
	client := iam.NewFromConfig(d.Config)
	err := deleteLoginProfile(ctx, client, d.Username)
	if err != nil {
		if !isNoSuchEntity(err) {
			logger.Error(fmt.Sprintf("Delete login profile failed: %s", err))
			return
		}
	}
	err = detachUserPolicy(ctx, client, d.Username)
	if err != nil {
		if !isNoSuchEntity(err) {
			logger.Error(fmt.Sprintf("Remove policy from %s failed: %s", d.Username, err))
			return
		}
	}
	err = deleteUser(ctx, client, d.Username)
	if err != nil {
		logger.Error(fmt.Sprintf("Delete user failed: %s", err))
		return
	}
	logger.Warning(fmt.Sprintf("Delete user %s success!", d.Username))
}

func detachUserPolicy(ctx context.Context, client *iam.Client, userName string) error {
	request := &iam.DetachUserPolicyInput{}
	request.UserName = &userName
	policyArn := "arn:aws:iam::aws:policy/AdministratorAccess"
	request.PolicyArn = &policyArn
	_, err := client.DetachUserPolicy(ctx, request)
	return err
}

func deleteLoginProfile(ctx context.Context, client *iam.Client, userName string) error {
	request := &iam.DeleteLoginProfileInput{}
	request.UserName = &userName
	_, err := client.DeleteLoginProfile(ctx, request)
	return err
}

func deleteUser(ctx context.Context, client *iam.Client, userName string) error {
	request := &iam.DeleteUserInput{}
	request.UserName = &userName
	_, err := client.DeleteUser(ctx, request)
	return err
}

func isNoSuchEntity(err error) bool {
	var target *iamtypes.NoSuchEntityException
	return errors.As(err, &target)
}
