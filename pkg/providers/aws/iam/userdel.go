package iam

import (
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/aws/aws-sdk-go/service/iam"
)

func (d *Driver) DelUser() {
	client := iam.New(d.Session)
	err := deleteLoginProfile(client, d.Username)
	if err != nil {
		if !strings.Contains(err.Error(), iam.ErrCodeNoSuchEntityException) {
			logger.Error(fmt.Sprintf("Delete login profile failed: %s", err))
			return
		}
	}
	err = detachUserPolicy(client, d.Username)
	if err != nil {
		if !strings.Contains(err.Error(), iam.ErrCodeNoSuchEntityException) {
			logger.Error(fmt.Sprintf("Remove policy from %s failed: %s", d.Username, err))
			return
		}
	}
	err = deleteUser(client, d.Username)
	if err != nil {
		logger.Error(fmt.Sprintf("Delete user failed: %s", err))
		return
	}
	logger.Warning(fmt.Sprintf("Delete user %s success!", d.Username))
}

func detachUserPolicy(client *iam.IAM, userName string) error {
	request := &iam.DetachUserPolicyInput{}
	request.UserName = &userName
	policyArn := "arn:aws:iam::aws:policy/AdministratorAccess"
	request.PolicyArn = &policyArn
	_, err := client.DetachUserPolicy(request)
	return err
}

func deleteLoginProfile(client *iam.IAM, userName string) error {
	request := &iam.DeleteLoginProfileInput{}
	request.UserName = &userName
	_, err := client.DeleteLoginProfile(request)
	return err
}

func deleteUser(client *iam.IAM, userName string) error {
	request := &iam.DeleteUserInput{}
	request.UserName = &userName
	_, err := client.DeleteUser(request)
	return err
}
