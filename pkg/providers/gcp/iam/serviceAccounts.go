package iam

import (
	"context"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/request"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Projects []string
	Token    string
}

func (d *Driver) GetServiceAccounts(ctx context.Context) ([]schema.User, error) {
	list := []schema.User{}
	logger.Info("Start enumerating IAM ...")
	r := &request.DefaultHttpRequest{
		Endpoint: "iam.googleapis.com",
		Method:   "GET",
		Token:    d.Token,
	}
	for _, project := range d.Projects {
		// https://console.cloud.google.com/apis/api/iam.googleapis.com/metrics
		accounts, err := r.ListServiceAccounts(project)
		if err != nil {
			logger.Error("List Service Accounts failed.")
			return list, err
		}

		for name, id := range accounts {
			_iam := schema.User{
				UserName: name,
				UserId:   id,
			}
			list = append(list, _iam)
		}

	}
	return list, nil
}
