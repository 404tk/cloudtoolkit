package iam

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/service/iam"
)

func (d *Driver) AddUser() {
	client := iam.New(d.Session)
	accountArn, err := createUser(client, d.Username)
	if err != nil {
		log.Println("[-] Create user failed:", err)
		if !strings.Contains(err.Error(), iam.ErrCodeEntityAlreadyExistsException) {
			return
		}
	}
	err = createLoginProfile(client, d.Username, d.Password)
	if err != nil {
		log.Println("[-] Create login password failed:", err)
		return
	}
	err = attachPolicyToUser(client, d.Username)
	if err != nil {
		log.Println("[-] Grant AdministratorAccess policy failed.")
		return
	}
	var url string
	if u := strings.Split(accountArn, ":"); len(u) > 4 {
		if u[1] == "aws-cn" {
			url = fmt.Sprintf("https://%s.signin.amazonaws.cn/console", u[4])
		} else {
			url = fmt.Sprintf("https://%s.signin.aws.amazon.com/console", u[4])
		}
	}
	fmt.Printf("\n%-10s\t%-20s\t%-60s\n", "Username", "Password", "Login URL")
	fmt.Printf("%-10s\t%-20s\t%-60s\n", "--------", "--------", "---------")
	fmt.Printf("%-10s\t%-20s\t%-60s\n\n", d.Username, d.Password, url)
}

func createUser(client *iam.IAM, userName string) (string, error) {
	resp, err := client.CreateUser(&iam.CreateUserInput{UserName: &userName})
	if err != nil {
		return "", err
	}
	return *resp.User.Arn, err
}

func createLoginProfile(client *iam.IAM, userName string, password string) error {
	request := &iam.CreateLoginProfileInput{}
	request.UserName = &userName
	request.Password = &password
	_, err := client.CreateLoginProfile(request)
	return err
}

func attachPolicyToUser(client *iam.IAM, userName string) error {
	request := &iam.AttachUserPolicyInput{}
	policyArn := "arn:aws:iam::aws:policy/AdministratorAccess"
	request.PolicyArn = &policyArn
	request.UserName = &userName
	_, err := client.AttachUserPolicy(request)
	return err
}
