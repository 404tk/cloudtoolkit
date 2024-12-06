package iam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/global"
	iam "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3/model"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3/region"
)

func (d *Driver) DelUser() {
	auth := global.NewCredentialsBuilder().
		WithAk(d.Auth.AK).
		WithSk(d.Auth.SK).
		Build()
	client := iam.NewIamClient(iam.IamClientBuilder().
		WithRegion(region.ValueOf("cn-north-1")).
		WithCredential(auth).
		Build())
	users, err := d.ListUsers(context.Background())
	if err != nil {
		logger.Error("List users failed:", err.Error())
		return
	}
	for _, u := range users {
		if u.UserName == d.Username {
			logger.Warning("Found UserId:", u.UserId)
			err := deleteUser(client, u.UserId)
			if err != nil {
				logger.Error(fmt.Sprintf("Delete user %s failed: %s", d.Username, err.Error()))
				return
			}
			logger.Warning(fmt.Sprintf("Delete user %s success!", d.Username))
			return
		}
	}
	logger.Error(fmt.Sprintf("User %s not found.", d.Username))
}

func deleteUser(client *iam.IamClient, uid string) error {
	_, err := client.KeystoneDeleteUser(&model.KeystoneDeleteUserRequest{UserId: uid})
	return err
}
