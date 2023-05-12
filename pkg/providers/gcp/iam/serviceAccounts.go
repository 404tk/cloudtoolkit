package iam

import (
	"context"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/request"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

type ServiceAccountProvider struct {
	Projects []string
	Token    string
}

func (d *ServiceAccountProvider) GetServiceAccounts(ctx context.Context) ([]*schema.User, error) {
	list := schema.NewResources().Users
	log.Println("[*] Start enumerating IAM ...")
	r := &request.DefaultHttpRequest{
		Endpoint: "iam.googleapis.com",
		Method:   "GET",
		Token:    d.Token,
	}
	for _, project := range d.Projects {
		// https://console.cloud.google.com/apis/api/iam.googleapis.com/metrics
		accounts, err := r.ListServiceAccounts(project)
		if err != nil {
			log.Println("[-] List Service Accounts failed.")
			return list, err
		}

		for name, id := range accounts {
			_iam := &schema.User{
				UserName: name,
				UserId:   id,
			}
			list = append(list, _iam)
		}

	}
	return list, nil
}
