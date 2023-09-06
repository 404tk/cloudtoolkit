package cam

import (
	"fmt"

	"github.com/404tk/cloudtoolkit/utils/logger"
	cam "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cam/v20190116"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

func (d *Driver) DelUser() {
	cpf := profile.NewClientProfile()
	client, _ := cam.NewClient(d.Credential, "", cpf)
	err := detachPolicyFromUser(client, d.UserName)
	if err != nil {
		logger.Error(fmt.Sprintf("Remove policy from %s failed: %s", d.UserName, err.Error()))
		return
	}
	err = deleteUser(client, d.UserName)
	if err != nil {
		logger.Error(fmt.Sprintf("Delete user %s failed: %s", d.UserName, err.Error()))
		return
	}
	logger.Info("Done.")
}

func detachPolicyFromUser(client *cam.Client, userName string) error {
	resp, err := getUserInfo(client, userName)
	if err != nil {
		return err
	}
	request := cam.NewDetachUserPolicyRequest()
	request.PolicyId = common.Uint64Ptr(1)
	request.DetachUin = resp.Response.Uin
	_, err = client.DetachUserPolicy(request)
	return err
}

func deleteUser(client *cam.Client, userName string) error {
	request := cam.NewDeleteUserRequest()
	request.Name = common.StringPtr(userName)
	request.Force = common.Uint64Ptr(1)
	_, err := client.DeleteUser(request)
	return err
}
