package cam

import (
	"log"

	cam "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cam/v20190116"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

func (d *CamUserProvider) DelUser() {
	cpf := profile.NewClientProfile()
	client, _ := cam.NewClient(d.Credential, "", cpf)
	err := detachPolicyFromUser(client, d.UserName)
	if err != nil {
		log.Printf("[-] Remove policy from %s failed: %s\n", d.UserName, err.Error())
		return
	}
	err = deleteUser(client, d.UserName)
	if err != nil {
		log.Printf("[-] Delete user %s failed: %s\n", d.UserName, err.Error())
		return
	}
}

func detachPolicyFromUser(client *cam.Client, userName string) error {
	resp, err := getUserInfo(client, userName)
	if err != nil {
		return err
	}
	request := cam.NewDetachUserPolicyRequest()
	request.PolicyId = common.Uint64Ptr(1)
	request.DetachUin = common.Uint64Ptr(*resp.Response.Uin)
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
