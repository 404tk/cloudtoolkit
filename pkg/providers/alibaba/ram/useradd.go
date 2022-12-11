package ram

import (
	"fmt"
	"log"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ram"
)

func (d *RamProvider) AddUser() {
	err := createUser(d.Client, d.UserName)
	if err != nil {
		log.Println("[-] Create user failed:", err.Error())
		return
	}
	err = createLoginProfile(d.Client, d.UserName, d.PassWord)
	if err != nil {
		log.Println("[-] Create login password failed:", err.Error())
		return
	}
	err = attachPolicyToUser(d.Client, d.UserName)
	if err != nil {
		log.Println("[-] Grant AdministratorAccess policy failed.")
		return
	}
	accountAlias := getAccountAlias(d.Client)
	fmt.Printf("\n%-10s\t%-10s\t%-60s\n", "Username", "Password", "Login URL")
	fmt.Printf("%-10s\t%-10s\t%-60s\n", "--------", "--------", "---------")
	fmt.Printf("%-10s\t%-10s\t%-60s\n\n",
		d.UserName, d.PassWord,
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
		log.Println("[-] Get account alias failed.")
		return ""
	}
	return response.AccountAlias
}
