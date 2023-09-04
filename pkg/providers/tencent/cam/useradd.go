package cam

import (
	"fmt"

	"github.com/404tk/cloudtoolkit/utils/logger"
	cam "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cam/v20190116"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

func (d *Driver) AddUser() {
	cpf := profile.NewClientProfile()
	client, _ := cam.NewClient(d.Credential, "", cpf)
	err := createUser(client, d.UserName, d.Password)
	if err != nil {
		logger.Error("Create user failed:", err.Error())
		return
	}
	err = attachPolicyToUser(client, d.UserName)
	OwnerID := getOwnerUin(client)
	fmt.Printf("\n%-10s\t%-10s\t%-60s\n", "Username", "Password", "Login URL")
	fmt.Printf("%-10s\t%-10s\t%-60s\n", "--------", "--------", "---------")
	fmt.Printf("%-10s\t%-10s\t%-60s\n\n",
		d.UserName,
		d.Password, "https://cloud.tencent.com/login/subAccount/"+OwnerID)
}

func createUser(client *cam.Client, userName, password string) error {
	request := cam.NewAddUserRequest()
	request.Name = common.StringPtr(userName)
	request.ConsoleLogin = common.Uint64Ptr(1)
	request.Password = common.StringPtr(password)
	request.NeedResetPassword = common.Uint64Ptr(0)
	_, err := client.AddUser(request)
	return err
}

func attachPolicyToUser(client *cam.Client, userName string) error {
	resp, err := getUserInfo(client, userName)
	if err != nil {
		return err
	}
	request := cam.NewAttachUserPolicyRequest()
	request.PolicyId = common.Uint64Ptr(1)
	request.AttachUin = resp.Response.Uin
	_, err = client.AttachUserPolicy(request)
	return err
}

func getUserInfo(client *cam.Client, userName string) (*cam.GetUserResponse, error) {
	request := cam.NewGetUserRequest()
	request.Name = common.StringPtr(userName)
	return client.GetUser(request)
}

func getOwnerUin(client *cam.Client) string {
	request := cam.NewGetUserAppIdRequest()
	response, err := client.GetUserAppId(request)
	if err != nil {
		logger.Error("Get user appid failed.")
		return ""
	}
	return *response.Response.OwnerUin
}
