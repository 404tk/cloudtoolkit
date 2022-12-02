package ram

import (
	"context"
	"log"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ram"
)

type RamProvider struct {
	Client *ram.Client
}

func (d *RamProvider) GetRamUser(ctx context.Context) ([]*schema.User, error) {
	list := schema.NewResources().Users
	marker := ""
	log.Println("[*] Start enumerating RAM ...")
	for {
		listUsersRequest := ram.CreateListUsersRequest()
		listUsersRequest.Scheme = "https"
		listUsersRequest.MaxItems = requests.NewInteger(100)
		listUsersRequest.Marker = marker
		response, err := d.Client.ListUsers(listUsersRequest)
		if err != nil {
			log.Println("[-] Enumerate RAM failed.")
			return list, err
		}

		for _, user := range response.Users.User {
			_user := schema.User{
				UserName: user.UserName,
				UserId:   user.UserId,
			}

			// 判断用户是否启用web控制台登录
			request := ram.CreateGetLoginProfileRequest()
			request.Scheme = "https"
			request.UserName = user.UserName
			_, err := d.Client.GetLoginProfile(request)
			if err == nil || err.(*errors.ServerError).Message() != "login policy not exists" {
				_user.EnableLogin = true
				getUserRequest := ram.CreateGetUserRequest()
				getUserRequest.Scheme = "https"
				getUserRequest.UserName = user.UserName
				getUserResponse, _ := d.Client.GetUser(getUserRequest)

				if getUserResponse.User.LastLoginDate != "" {
					lastLoginDate, _ := time.Parse(time.RFC3339, getUserResponse.User.LastLoginDate)
					_user.LastLogin = lastLoginDate.String()
				}
			}

			list = append(list, &_user)
		}
		if !response.IsTruncated {
			break
		}
		marker = response.Marker
	}
	return list, nil
}
