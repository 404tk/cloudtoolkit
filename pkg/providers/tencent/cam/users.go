package cam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
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

func (d *Driver) ListUsers(ctx context.Context) ([]schema.User, error) {
	list := []schema.User{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List CAM users ...")
	}
	cpf := profile.NewClientProfile()
	// cpf.HttpProfile.Endpoint = "cam.tencentcloudapi.com"
	client, err := cam.NewClient(d.Credential, "", cpf)
	if err != nil {
		return list, err
	}
	listUsersRequest := cam.NewListUsersRequest()
	listUsersResponse, err := client.ListUsers(listUsersRequest)
	if err != nil {
		logger.Error("List users failed.")
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
		if utils.ListPolicies {
			_user.Policies = listAttachedUserAllPolicies(client, user.Uin)
		}

		list = append(list, _user)
	}
	return list, nil
}
