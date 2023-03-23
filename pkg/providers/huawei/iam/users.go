package iam

import (
	"context"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/global"
	iam "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3/model"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3/region"
)

type IAMUserProvider struct {
	Auth     basic.Credentials
	Regions  []string
	Username string
	Password string
}

func (d *IAMUserProvider) GetIAMUser(ctx context.Context) ([]*schema.User, error) {
	list := schema.NewResources().Users
	select {
	case <-ctx.Done():
		return list, nil
	default:
		log.Println("[*] Start enumerating IAM user ...")
	}
	auth := global.NewCredentialsBuilder().
		WithAk(d.Auth.AK).
		WithSk(d.Auth.SK).
		Build()
	client := iam.NewIamClient(iam.IamClientBuilder().
		WithRegion(region.ValueOf(d.Regions[0])).
		WithCredential(auth).
		Build())
	keystoneListUsersRequest := &model.KeystoneListUsersRequest{}
	keystoneListUsersResponse, err := client.KeystoneListUsers(keystoneListUsersRequest)
	if err != nil {
		log.Println("[-] Enumerate IAM failed.")
		return list, err
	}

	for _, user := range *keystoneListUsersResponse.Users {
		_user := &schema.User{
			UserName:    user.Name,
			UserId:      user.Id,
			EnableLogin: user.Enabled,
		}
		list = append(list, _user)
	}

	return list, nil
}
