package iam

import (
	"fmt"
	"log"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/global"
	iam "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3/model"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3/region"
)

func (d *IAMUserProvider) AddUser() {
	auth := global.NewCredentialsBuilder().
		WithAk(d.Auth.AK).
		WithSk(d.Auth.SK).
		Build()
	client := iam.NewIamClient(iam.IamClientBuilder().
		WithRegion(region.ValueOf(d.Regions[0])).
		WithCredential(auth).
		Build())
	uid, domainid, err := createUser(client, d.Username, d.Password)
	if err != nil {
		log.Println("[-] Create user failed:", err.Error())
		return
	}
	err = addUserToAdminGroup(client, uid)
	if err != nil {
		log.Println("[-] Grant AdministratorAccess policy failed.")
		return
	}
	name := getDomainName(client, domainid)
	fmt.Printf("\n%-10s\t%-10s\t%-60s\n", "Username", "Password", "Login URL")
	fmt.Printf("%-10s\t%-10s\t%-60s\n", "--------", "--------", "---------")
	fmt.Printf("%-10s\t%-10s\t%-60s\n\n",
		d.Username,
		d.Password, "https://auth.huaweicloud.com/authui/login?id="+name)
}

func createUser(client *iam.IamClient, uname, pwd string) (string, string, error) {
	enable := true
	request := &model.KeystoneCreateUserRequest{
		Body: &model.KeystoneCreateUserRequestBody{
			User: &model.KeystoneCreateUserOption{
				Name:     uname,
				Password: &pwd,
				Enabled:  &enable,
			}}}
	resp, err := client.KeystoneCreateUser(request)
	if err != nil {
		return "", "", err
	}
	return resp.User.Id, resp.User.DomainId, err
}

func addUserToAdminGroup(client *iam.IamClient, uid string) error {
	request := &model.KeystoneListGroupsRequest{}
	resp, err := client.KeystoneListGroups(request)
	if err != nil {
		return err
	}

	groups := make(map[string]string)
	for _, v := range *resp.Groups {
		groups[v.Name] = v.Id
	}

	if g, ok := groups["admin"]; ok {
		_, err = client.KeystoneAddUserToGroup(
			&model.KeystoneAddUserToGroupRequest{
				GroupId: g,
				UserId:  uid,
			})
	} else {
		for _, g := range groups {
			_, err = client.KeystoneAddUserToGroup(
				&model.KeystoneAddUserToGroupRequest{
					GroupId: g,
					UserId:  uid,
				})
		}
	}
	return err
}

func getDomainName(client *iam.IamClient, domainid string) string {
	resp, err := client.KeystoneListAuthDomains(&model.KeystoneListAuthDomainsRequest{})
	if err != nil {
		log.Println("[-] List domains failed:", err.Error())
		return ""
	}
	for _, v := range *resp.Domains {
		if v.Id == domainid {
			return v.Name
		}
	}
	return ""
}
