package ram

import (
	"fmt"
	"log"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ram"
)

func (d *RamProvider) AddRole() {
	err := createRole(d.Client, d.RoleName, d.AccountId)
	if err != nil {
		log.Println("[-] Create role failed:", err.Error())
		return
	}
	err = attachPolicyToRole(d.Client, d.RoleName)
	if err != nil {
		log.Println("[-] Grant AdministratorAccess policy failed.")
		return
	}
	accountAlias := getAccountAlias(d.Client)
	fmt.Printf("\n%-20s\t%-10s\t%-60s\n", "AccountAlias", "RoleName", "Switch URL")
	fmt.Printf("%-20s\t%-10s\t%-60s\n", "------------", "--------", "----------")
	fmt.Printf("%-20s\t%-10s\t%-60s\n\n",
		accountAlias, d.RoleName,
		"https://signin.aliyun.com/switchRole.htm")
}

func createRole(client *ram.Client, roleName, accountId string) error {
	request := ram.CreateCreateRoleRequest()
	request.Scheme = "https"
	request.RoleName = roleName
	request.AssumeRolePolicyDocument = fmt.Sprintf(
		"{\"Statement\":[{\"Action\":\"sts:AssumeRole\",\"Effect\":\"Allow\",\"Principal\":{\"RAM\":\"acs:ram::%s:root\"}}],\"Version\":\"1\"}",
		accountId)
	_, err := client.CreateRole(request)
	return err
}

func attachPolicyToRole(client *ram.Client, roleName string) error {
	request := ram.CreateAttachPolicyToRoleRequest()
	request.Scheme = "https"
	request.PolicyType = "System"
	request.PolicyName = "AdministratorAccess"
	request.RoleName = roleName
	_, err := client.AttachPolicyToRole(request)
	return err
}
