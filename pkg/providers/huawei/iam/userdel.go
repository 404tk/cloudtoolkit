package iam

import (
	"context"
	"fmt"
	"net/http"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func (d *Driver) DelUser() {
	users, err := d.ListUsers(context.Background())
	if err != nil {
		logger.Error("List users failed:", err.Error())
		return
	}
	for _, u := range users {
		if u.UserName == d.Username {
			logger.Warning("Found UserId:", u.UserId)
			err := d.deleteUser(context.Background(), u.UserId)
			if err != nil {
				logger.Error(fmt.Sprintf("Delete user %s failed: %s", d.Username, err.Error()))
				return
			}
			logger.Warning(fmt.Sprintf("Delete user %s success!", d.Username))
			return
		}
	}
	logger.Error(fmt.Sprintf("User %s not found.", d.Username))
}

func (d *Driver) deleteUser(ctx context.Context, uid string) error {
	region, err := d.requestRegion()
	if err != nil {
		return err
	}
	return d.client().DoJSON(ctx, api.Request{
		Service: "iam",
		Region:  region,
		Intl:    d.Cred.Intl,
		Method:  http.MethodDelete,
		Path:    fmt.Sprintf("/v3/users/%s", uid),
		Headers: d.domainHeaders(ctx),
	}, nil)
}
