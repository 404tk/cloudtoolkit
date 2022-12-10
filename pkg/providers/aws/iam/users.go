package iam

import (
	"context"
	"log"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

type IAMProvider struct {
	Session *session.Session
}

func (d *IAMProvider) GetIAMUser(ctx context.Context) ([]*schema.User, error) {
	list := schema.NewResources().Users
	log.Println("[*] Start enumerating IAM ...")
	client := iam.New(d.Session)
	users, err := client.ListUsers(&iam.ListUsersInput{})
	if err != nil {
		log.Println("[-] List users failed.")
		return list, err
	}
	for _, user := range users.Users {
		_user := &schema.User{
			UserName:   *user.UserName,
			UserId:     *user.UserId,
			CreateTime: user.CreateDate.Format(time.RFC3339),
		}
		if user.PasswordLastUsed != nil {
			_user.LastLogin = user.PasswordLastUsed.Format(time.RFC3339)
			_user.EnableLogin = true
		}
		list = append(list, _user)
	}

	return list, nil
}
