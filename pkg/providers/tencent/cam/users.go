package cam

import (
	"context"
	"log"
	"strconv"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	cam "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cam/v20190116"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

type CamUserProvider struct {
	Credential *common.Credential
	UserName   string
	Password   string
}

func (d *CamUserProvider) GetCamUser(ctx context.Context) ([]*schema.User, error) {
	list := schema.NewResources().Users
	log.Println("[*] Start enumerating CAM ...")
	cpf := profile.NewClientProfile()
	// cpf.HttpProfile.Endpoint = "cam.tencentcloudapi.com"
	client, _ := cam.NewClient(d.Credential, "", cpf)
	listUsersRequest := cam.NewListUsersRequest()
	listUsersResponse, err := client.ListUsers(listUsersRequest)
	if err != nil {
		log.Println("[-] Enumerate CAM failed.")
		return list, err
	}
	for _, user := range listUsersResponse.Response.Data {
		_user := &schema.User{
			UserName: *user.Name,
			UserId:   strconv.FormatUint(*user.Uid, 10),
		}
		if *user.ConsoleLogin == 1 {
			_user.EnableLogin = true
			_user.CreateTime = *user.CreateTime
		}

		list = append(list, _user)
	}
	return list, nil
}
