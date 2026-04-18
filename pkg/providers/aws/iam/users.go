package iam

import (
	"context"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

type Driver struct {
	Config   awsv2.Config
	Username string
	Password string
}

func (d *Driver) ListUsers(ctx context.Context) ([]schema.User, error) {
	list := []schema.User{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List IAM users ...")
	}
	client := iam.NewFromConfig(d.Config)
	paginator := iam.NewListUsersPaginator(client, &iam.ListUsersInput{})
	for paginator.HasMorePages() {
		users, err := paginator.NextPage(ctx)
		if err != nil {
			logger.Error("List users failed.")
			return list, err
		}
		for _, user := range users.Users {
			createTime := ""
			if user.CreateDate != nil {
				createTime = user.CreateDate.Format(time.RFC3339)
			}
			_user := schema.User{
				UserName:   awsv2.ToString(user.UserName),
				UserId:     awsv2.ToString(user.UserId),
				CreateTime: createTime,
			}
			if user.PasswordLastUsed != nil {
				_user.LastLogin = user.PasswordLastUsed.Format(time.RFC3339)
				_user.EnableLogin = true
			} else {
				req := &iam.GetLoginProfileInput{UserName: user.UserName}
				lp, err := client.GetLoginProfile(ctx, req)
				if err == nil && lp.LoginProfile != nil {
					_user.EnableLogin = true
				}
			}
			_user.Policies = listAttachedUserPolicies(ctx, client, _user.UserName)
			list = append(list, _user)
		}
	}

	return list, nil
}
