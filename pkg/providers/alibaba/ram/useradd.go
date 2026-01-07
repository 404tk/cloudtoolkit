package ram

import (
	"fmt"

	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ram"
)

func (d *Driver) AddUser() {
	client, err := d.NewClient()
	if err != nil {
		logger.Error("Create RAM client failed:", err.Error())
		return
	}
	err = createUser(client, d.UserName)
	if err != nil {
		logger.Error("Create user failed:", err.Error())
		return
	}
	err = createLoginProfile(client, d.UserName, d.Password)
	if err != nil {
		logger.Error("Create login password failed:", err.Error())
		return
	}
	err = attachPolicyToUser(client, d.UserName)
	if err != nil {
		logger.Error("Grant AdministratorAccess policy failed.")
		return
	}
	accountAlias := getAccountAlias(client)
	fmt.Printf("\n%-10s\t%-10s\t%-60s\n", "Username", "Password", "Login URL")
	fmt.Printf("%-10s\t%-10s\t%-60s\n", "--------", "--------", "---------")
	fmt.Printf("%-10s\t%-10s\t%-60s\n\n",
		d.UserName, d.Password,
		fmt.Sprintf("https://signin.aliyun.com/%s/login.htm", accountAlias))
}

func createUser(client *ram.Client, userName string) error {
	request := ram.CreateCreateUserRequest()
	request.Scheme = "https"
	request.UserName = userName
	_, err := client.CreateUser(request)
	return err
}

func createLoginProfile(client *ram.Client, userName string, password string) error {
	request := ram.CreateCreateLoginProfileRequest()
	request.Scheme = "https"
	request.UserName = userName
	request.Password = password
	_, err := client.CreateLoginProfile(request)
	return err
}

func attachPolicyToUser(client *ram.Client, userName string) error {
	request := ram.CreateAttachPolicyToUserRequest()
	request.Scheme = "https"
	request.PolicyType = "System"
	request.PolicyName = "AdministratorAccess"
	request.UserName = userName
	_, err := client.AttachPolicyToUser(request)
	return err
}

func getAccountAlias(client *ram.Client) string {
	request := ram.CreateGetAccountAliasRequest()
	request.Scheme = "https"
	response, err := client.GetAccountAlias(request)
	if err != nil {
		logger.Error("Get account alias failed.")
		return ""
	}
	return response.AccountAlias
}
