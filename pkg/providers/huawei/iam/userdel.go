package iam

import (
	"context"
	"log"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/global"
	iam "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3/model"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3/region"
)

func (d *Driver) DelUser() {
	auth := global.NewCredentialsBuilder().
		WithAk(d.Auth.AK).
		WithSk(d.Auth.SK).
		Build()
	client := iam.NewIamClient(iam.IamClientBuilder().
		WithRegion(region.ValueOf(d.Regions[0])).
		WithCredential(auth).
		Build())
	users, err := d.GetIAMUser(context.Background())
	if err != nil {
		log.Println("[-] List users failed:", err.Error())
		return
	}
	for _, u := range users {
		if u.UserName == d.Username {
			log.Println("[+] Found UserId:", u.UserId)
			err := deleteUser(client, u.UserId)
			if err != nil {
				log.Printf("[-] Delete user %s failed: %s\n", d.Username, err.Error())
				return
			}
			log.Printf("[+] Delete user %s success!\n", d.Username)
			return
		}
	}
	log.Printf("[-] User %s not found.\n", d.Username)
}

func deleteUser(client *iam.IamClient, uid string) error {
	_, err := client.KeystoneDeleteUser(&model.KeystoneDeleteUserRequest{UserId: uid})
	return err
}
