package iam

import (
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/service/iam"
)

func (d *Driver) DelUser() {
	client := iam.New(d.Session)
	err := deleteLoginProfile(client, d.Username)
	if err != nil {
		log.Printf("[-] Delete login profile failed: %s\n", err)
		if !strings.Contains(err.Error(), iam.ErrCodeNoSuchEntityException) {
			return
		}
	}
	err = detachUserPolicy(client, d.Username)
	if err != nil {
		log.Printf("[-] Remove policy from %s failed: %s\n", d.Username, err)
		if !strings.Contains(err.Error(), iam.ErrCodeNoSuchEntityException) {
			return
		}
	}
	err = deleteUser(client, d.Username)
	if err != nil {
		log.Printf("[-] Delete user failed: %s\n", err)
		return
	}
	log.Printf("[+] Delete user %s success!\n", d.Username)
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
