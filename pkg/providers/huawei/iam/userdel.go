package iam

import (
	"context"
	"fmt"
	"net/http"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

func (d *Driver) DelUser() (schema.IAMResult, error) {
	users, err := d.ListUsers(context.Background())
	if err != nil {
		return schema.IAMResult{}, fmt.Errorf("list users failed: %w", err)
	}
	for _, u := range users {
		if u.UserName == d.Username {
			err := d.deleteUser(context.Background(), u.UserId)
			if err != nil {
				return schema.IAMResult{}, fmt.Errorf("delete user %s failed: %w", d.Username, err)
			}
			return schema.IAMResult{
				Username: d.Username,
				Message:  "User deleted successfully",
			}, nil
		}
	}
	return schema.IAMResult{}, fmt.Errorf("user %s not found", d.Username)
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
