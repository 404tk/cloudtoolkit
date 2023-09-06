package ram

import (
	"fmt"

	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ram"
)

func (d *Driver) DelRole() {
	client := d.NewClient()
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
	logger.Warning("Done.")
}

func detachPolicyFromRole(client *ram.Client, roleName string) error {
	request := ram.CreateDetachPolicyFromRoleRequest()
	request.Scheme = "https"
	request.PolicyType = "System"
	request.PolicyName = "AdministratorAccess"
	request.RoleName = roleName
	_, err := client.DetachPolicyFromRole(request)
	return err
}

func deleteRole(client *ram.Client, roleName string) error {
	request := ram.CreateDeleteRoleRequest()
	request.Scheme = "https"
	request.RoleName = roleName
	_, err := client.DeleteRole(request)
	return err
}
