package iam

import (
	"context"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ram"
)

type Driver struct {
	Cred      *credentials.StsTokenCredential
	Region    string
	UserName  string
	Password  string
	RoleName  string
	AccountId string
}

func (d *Driver) NewClient() (*ram.Client, error) {
	region := d.Region
	if region == "all" {
		region = "cn-hangzhou"
	}
	return ram.NewClientWithOptions(region, sdk.NewConfig(), d.Cred)
}

func (d *Driver) ListUsers(ctx context.Context) ([]schema.User, error) {
	list := []schema.User{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List RAM users ...")
	}
	client, err := d.NewClient()
	if err != nil {
		return list, err
	}
	marker := ""
	policy_infos = make(map[string]string)
	for {
		listUsersRequest := ram.CreateListUsersRequest()
		listUsersRequest.Scheme = "https"
		listUsersRequest.MaxItems = requests.NewInteger(100)
		listUsersRequest.Marker = marker
		response, err := client.ListUsers(listUsersRequest)
		if err != nil {
			logger.Error("List users failed.")
			return list, err
		}

		for _, user := range response.Users.User {
			_user := schema.User{
				UserName: user.UserName,
				UserId:   user.UserId,
			}
			date, _ := time.Parse(time.RFC3339, user.CreateDate)
			_user.CreateTime = date.String()

			request := ram.CreateGetLoginProfileRequest()
			request.Scheme = "https"
			request.UserName = user.UserName
			_, err := client.GetLoginProfile(request)
			if err == nil || err.(*errors.ServerError).Message() != "login policy not exists" {
				_user.EnableLogin = true
				getUserRequest := ram.CreateGetUserRequest()
				getUserRequest.Scheme = "https"
				getUserRequest.UserName = user.UserName
				getUserResponse, err := client.GetUser(getUserRequest)
				if err == nil && getUserResponse.User.LastLoginDate != "" {
					lastLoginDate, _ := time.Parse(time.RFC3339, getUserResponse.User.LastLoginDate)
					_user.LastLogin = lastLoginDate.String()
				}
			}

			if utils.ListPolicies {
				_user.Policies = listPoliciesForUser(client, _user.UserName)
			}

			list = append(list, _user)
			select {
			case <-ctx.Done():
				return list, nil
			default:
				continue
			}
		}
		if !response.IsTruncated {
			break
		}
		marker = response.Marker
	}
	return list, nil
}
