package iam

import (
	"context"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"google.golang.org/api/iam/v1"
)

type ServiceAccountProvider struct {
	IamService *iam.Service
	Projects   []string
}

func (d *ServiceAccountProvider) GetServiceAccounts(ctx context.Context) ([]*schema.User, error) {
	list := schema.NewResources().Users
	log.Println("[*] Start enumerating IAM ...")
	for _, project := range d.Projects {
		// https://console.cloud.google.com/apis/api/iam.googleapis.com/metrics
		accounts, err := d.IamService.Projects.ServiceAccounts.List("projects/" + project).Do()
		if err != nil {
			log.Println("[-] List Service Accounts failed.")
			return list, err
		}

		for _, account := range accounts.Accounts {
			_iam := &schema.User{
				UserName: account.DisplayName,
				UserId:   account.UniqueId,
			}
			list = append(list, _iam)
		}
	}
	return list, nil
}
