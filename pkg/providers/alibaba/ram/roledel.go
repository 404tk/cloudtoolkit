package ram

import (
	"log"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ram"
)

func (d *RamProvider) DelRole() {
	client := d.NewClient()
	err := detachPolicyFromRole(client, d.RoleName)
	if err != nil {
		log.Printf("[-] Remove policy from %s failed: %s\n", d.RoleName, err.Error())
		return
	}
	err = deleteRole(client, d.RoleName)
	if err != nil {
		log.Printf("[-] Delete role %s failed: %s\n", d.RoleName, err.Error())
		return
	}
	log.Println("[+] Done.")
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
