package cam

import (
	"fmt"

	"github.com/404tk/cloudtoolkit/utils/logger"
	cam "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cam/v20190116"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

func (d *Driver) DelRole() {
	cpf := profile.NewClientProfile()
	client, _ := cam.NewClient(d.Credential, "", cpf)
	err := detachPolicyFromRole(client, d.RoleName)
	if err != nil {
		logger.Error(fmt.Sprintf("Remove policy from %s failed: %s", d.RoleName, err.Error()))
		return
	}
	err = deleteRole(client, d.RoleName)
	if err != nil {
		logger.Error(fmt.Sprintf("Delete role %s failed: %s", d.RoleName, err.Error()))
		return
	}
	logger.Info("Done.")
}

func detachPolicyFromRole(client *cam.Client, roleName string) error {
	request := cam.NewDetachRolePolicyRequest()
	request.PolicyId = common.Uint64Ptr(1)
	request.DetachRoleName = common.StringPtr(roleName)
	_, err := client.DetachRolePolicy(request)
	return err
}

func deleteRole(client *cam.Client, roleName string) error {
	request := cam.NewDeleteRoleRequest()
	request.RoleName = common.StringPtr(roleName)
	_, err := client.DeleteRole(request)
	return err
}
