package cam

import (
	"fmt"
	"log"

	cam "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cam/v20190116"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

func (d *CamUserProvider) AddRole() {
	cpf := profile.NewClientProfile()
	client, _ := cam.NewClient(d.Credential, "", cpf)
	err := createRole(client, d.RoleName, d.Uin)
	if err != nil {
		log.Println("[-] Create role failed:", err.Error())
		return
	}
	err = attachPolicyToRole(client, d.RoleName)
	OwnerID := getOwnerUin(client)
	log.Printf("[+] Switch URL: https://cloud.tencent.com/cam/switchrole?ownerUin=%s&roleName=%s\n", OwnerID, d.RoleName)
}

func createRole(client *cam.Client, roleName, uin string) error {
	request := cam.NewCreateRoleRequest()
	request.RoleName = common.StringPtr(roleName)
	request.ConsoleLogin = common.Uint64Ptr(1)
	request.SessionDuration = common.Uint64Ptr(10000)
	policy := fmt.Sprintf(
		`{"version":"2.0","statement":[{"action":"name/sts:AssumeRole","effect":"allow","principal":{"qcs":["qcs::cam::uin/%s:root"]}}]}`, uin)
	request.PolicyDocument = common.StringPtr(policy)
	_, err := client.CreateRole(request)
	return err
}

func attachPolicyToRole(client *cam.Client, roleName string) error {
	request := cam.NewAttachRolePolicyRequest()
	request.PolicyId = common.Uint64Ptr(1)
	request.AttachRoleName = common.StringPtr(roleName)
	_, err := client.AttachRolePolicy(request)
	return err
}
