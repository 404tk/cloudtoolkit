package ram

import (
	"fmt"

	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ram"
)

func (d *Driver) AddRole() {
	client := d.NewClient()
	err := createRole(client, d.RoleName, d.AccountId)
	if err != nil {
		logger.Error("Create role failed:", err.Error())
		return
	}
	err = attachPolicyToRole(client, d.RoleName)
	if err != nil {
		logger.Error("Grant AdministratorAccess policy failed.")
		return
	}
	accountAlias := getAccountAlias(client)
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
