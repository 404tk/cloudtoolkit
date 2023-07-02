package ram

import (
	"context"
	"log"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ram"
)

type RamProvider struct {
	Cred      *credentials.StsTokenCredential
	Region    string
	UserName  string
	PassWord  string
	RoleName  string
	AccountId string
}

func (d *RamProvider) NewClient() *ram.Client {
	region := d.Region
	if region == "all" {
		region = "cn-hangzhou"
	}
	client, _ := ram.NewClientWithOptions(region, sdk.NewConfig(), d.Cred)
	return client
}

func (d *RamProvider) GetRamUser(ctx context.Context) ([]*schema.User, error) {
	list := schema.NewResources().Users
	select {
	case <-ctx.Done():
		return list, nil
	default:
		log.Println("[*] Start enumerating RAM ...")
	}
	client := d.NewClient()
	marker := ""
	for {
		listUsersRequest := ram.CreateListUsersRequest()
		listUsersRequest.Scheme = "https"
		listUsersRequest.MaxItems = requests.NewInteger(100)
		listUsersRequest.Marker = marker
		response, err := client.ListUsers(listUsersRequest)
		if err != nil {
			log.Println("[-] Enumerate RAM failed.")
			return list, err
		}

		for _, user := range response.Users.User {
			_user := schema.User{
				UserName: user.UserName,
				UserId:   user.UserId,
			}

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

			list = append(list, &_user)
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
