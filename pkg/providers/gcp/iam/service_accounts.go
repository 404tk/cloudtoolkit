package iam

import (
	"context"
	"net/http"
	"net/url"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Projects []string
	Client   *api.Client
}

func (d *Driver) ListUsers(ctx context.Context) ([]schema.User, error) {
	list := []schema.User{}
	logger.Info("List IAM users ...")
	for _, project := range d.Projects {
		accounts, err := d.listServiceAccounts(ctx, project)
		if err != nil {
			logger.Error("List Service Accounts failed.")
			return list, err
		}

		for _, account := range accounts {
			if account.DisplayName == "" {
				continue
			}
			_iam := schema.User{
				UserName: account.DisplayName,
				UserId:   account.UniqueID,
			}
			list = append(list, _iam)
		}

	}
	return list, nil
}

func (d *Driver) listServiceAccounts(ctx context.Context, project string) ([]api.ServiceAccount, error) {
	pager := api.NewPager[api.ServiceAccount](d.Client, api.Request{
		Method:  http.MethodGet,
		BaseURL: api.IAMBaseURL,
		Path:    "/v1/projects/" + url.PathEscape(project) + "/serviceAccounts",
		Query: url.Values{
			"pageSize": {"100"},
		},
		Idempotent: true,
	}, "accounts")
	return pager.All(ctx)
}
