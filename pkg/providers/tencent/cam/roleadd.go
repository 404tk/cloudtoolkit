package cam

import (
	"fmt"

	"github.com/404tk/cloudtoolkit/utils/logger"
	cam "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cam/v20190116"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

func (d *Driver) AddRole() {
	cpf := profile.NewClientProfile()
	client, _ := cam.NewClient(d.Credential, "", cpf)
	err := createRole(client, d.RoleName, d.Uin)
	if err != nil {
		logger.Error("Create role failed:", err.Error())
		return
	}
	_ = attachPolicyToRole(client, d.RoleName)
	OwnerID := getOwnerUin(client)
	logger.Warning(fmt.Sprintf(
		"Switch URL: https://cloud.tencent.com/cam/switchrole?ownerUin=%s&roleName=%s\n",
		OwnerID, d.RoleName))
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
