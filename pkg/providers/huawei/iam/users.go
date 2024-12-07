package iam

import (
	"context"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/global"
	iam "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3/model"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3/region"
)

type Driver struct {
	Auth     basic.Credentials
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
	auth := global.NewCredentialsBuilder().
		WithAk(d.Auth.AK).
		WithSk(d.Auth.SK).
		Build()
	client := iam.NewIamClient(iam.IamClientBuilder().
		WithRegion(region.ValueOf("cn-north-1")).
		WithCredential(auth).
		Build())
	keystoneListUsersRequest := &model.KeystoneListUsersRequest{}
	keystoneListUsersResponse, err := client.KeystoneListUsers(keystoneListUsersRequest)
	if err != nil {
		logger.Error("List users failed.")
		return list, err
	}

	for _, user := range *keystoneListUsersResponse.Users {
		_user := schema.User{
			UserName:    user.Name,
			UserId:      user.Id,
			EnableLogin: user.Enabled,
		}
		list = append(list, _user)
	}

	return list, nil
}
