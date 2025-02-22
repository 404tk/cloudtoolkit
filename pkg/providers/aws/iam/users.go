package iam

import (
	"context"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

type Driver struct {
	Session  *session.Session
	Username string
	Password string
}

func (d *Driver) GetIAMUser(ctx context.Context) ([]schema.User, error) {
	list := []schema.User{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List IAM users ...")
	}
	client := iam.New(d.Session)
	users, err := client.ListUsers(&iam.ListUsersInput{})
	if err != nil {
		logger.Error("List users failed.")
		return list, err
	}
	for _, user := range users.Users {
		_user := schema.User{
			UserName:   *user.UserName,
			UserId:     *user.UserId,
			CreateTime: user.CreateDate.Format(time.RFC3339),
		}
		if user.PasswordLastUsed != nil {
			_user.LastLogin = user.PasswordLastUsed.Format(time.RFC3339)
			_user.EnableLogin = true
		} else {
			req := &iam.GetLoginProfileInput{UserName: user.UserName}
			lp, err := client.GetLoginProfile(req)
			if err == nil && lp.LoginProfile != nil {
				_user.EnableLogin = true
			}
		}
		_user.Policies = listAttachedUserPolicies(client, _user.UserName)
		list = append(list, _user)
	}

	return list, nil
}
