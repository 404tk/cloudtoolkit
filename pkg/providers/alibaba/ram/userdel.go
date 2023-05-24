package ram

import (
	"log"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ram"
)

func (d *RamProvider) DelUser() {
	err := detachPolicyFromUser(d.Client, d.UserName)
	if err != nil {
		log.Printf("[-] Remove policy from %s failed: %s\n", d.UserName, err.Error())
		return
	}
	err = deleteUser(d.Client, d.UserName)
	if err != nil {
		log.Printf("[-] Delete user %s failed: %s\n", d.UserName, err.Error())
		return
	}
	log.Println("[+] Done.")
}

func detachPolicyFromUser(client *ram.Client, userName string) error {
	request := ram.CreateDetachPolicyFromUserRequest()
	request.Scheme = "https"
	request.PolicyType = "System"
	request.PolicyName = "AdministratorAccess"
	request.UserName = userName
	_, err := client.DetachPolicyFromUser(request)
	return err
}

func deleteUser(client *ram.Client, userName string) error {
	request := ram.CreateDeleteUserRequest()
	request.Scheme = "https"
	request.UserName = userName
	_, err := client.DeleteUser(request)
	return err
}
