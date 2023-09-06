package cam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	cam "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cam/v20190116"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

type Driver struct {
	Credential *common.Credential
	UserName   string
	Password   string
	RoleName   string
	Uin        string
}

func (d *Driver) GetCamUser(ctx context.Context) ([]schema.User, error) {
	list := schema.NewResources().Users
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("Start enumerating CAM ...")
	}
	cpf := profile.NewClientProfile()
	// cpf.HttpProfile.Endpoint = "cam.tencentcloudapi.com"
	client, _ := cam.NewClient(d.Credential, "", cpf)
	listUsersRequest := cam.NewListUsersRequest()
	listUsersResponse, err := client.ListUsers(listUsersRequest)
	if err != nil {
		logger.Error("Enumerate CAM failed.")
		return list, err
	}
	policy_infos = make(map[string]string)
	for _, user := range listUsersResponse.Response.Data {
		_user := schema.User{
			UserName:   *user.Name,
			UserId:     fmt.Sprintf("%v", *user.Uin),
			CreateTime: *user.CreateTime,
		}
		if *user.ConsoleLogin == 1 {
			_user.EnableLogin = true
		}
		_user.Policies = listAttachedUserAllPolicies(client, user.Uin)

		list = append(list, _user)
	}
	return list, nil
}
