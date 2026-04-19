package iam

import (
	"context"
	"net/http"
	"net/url"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func (d *Driver) ListUsers(ctx context.Context) ([]schema.User, error) {
	list := []schema.User{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List IAM users ...")
	}
	region, err := d.requestRegion()
	if err != nil {
		logger.Error("List users failed.")
		return list, err
	}
	query := url.Values{}
	var resp api.ListUsersV5Response
	err = d.client().DoJSON(ctx, api.Request{
		Service:    "iam",
		Region:     region,
		Intl:       d.Cred.Intl,
		Method:     http.MethodGet,
		Path:       "/v5/users",
		Query:      query,
		Idempotent: true,
	}, &resp)
	if err != nil {
		logger.Error("List users failed.")
		return list, err
	}

	for _, user := range resp.Users {
		_user := schema.User{
			UserName:    user.UserName,
			UserId:      user.UserID,
			EnableLogin: user.Enabled,
		}
		list = append(list, _user)
	}

	return list, nil
}
