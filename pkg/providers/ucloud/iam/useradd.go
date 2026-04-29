package iam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

const ucloudConsoleURL = "https://passport.ucloud.cn/login/subAccount/%d"

func (d *Driver) AddUser() (schema.IAMResult, error) {
	ctx := context.Background()

	client := d.client()
	var resp api.IAMCreateUserResponse
	err := client.Do(ctx, api.Request{
		Action: "CreateUser",
		Params: d.actionParams(map[string]any{
			"UserName":           d.UserName,
			"DisplayName":        d.UserName,
			"AccessKeyStatus":    "Inactive",
			"LoginProfileStatus": "Active",
			"Password":           d.Password,
		}),
	}, &resp)
	if err != nil {
		return schema.IAMResult{}, fmt.Errorf("create user failed: %w", err)
	}

	if err := d.grantProjectAccess(ctx, client, d.UserName); err != nil {
		_, _ = d.deleteUser(ctx, client, d.UserName)
		return schema.IAMResult{}, fmt.Errorf("grant project access failed: %w", err)
	}
	if err := d.grantIamAccess(ctx, client, d.UserName); err != nil {
		_, _ = d.deleteUser(ctx, client, d.UserName)
		return schema.IAMResult{}, fmt.Errorf("grant iam access failed: %w", err)
	}

	return schema.IAMResult{
		Username:  resp.UserName,
		Password:  resp.Password,
		LoginURL:  fmt.Sprintf(ucloudConsoleURL, resp.CompanyID),
		AccountID: fmt.Sprint(resp.CompanyID),
		Message:   "User created successfully with project access enabled",
	}, nil
}

func (d *Driver) grantProjectAccess(ctx context.Context, client *api.Client, userName string) error {
	var resp api.IAMAttachPoliciesToUserResponse
	err := client.Do(ctx, api.Request{
		Action: "AttachPoliciesToUser",
		Params: d.actionParams(map[string]any{
			"UserName":   userName,
			"PolicyURNs": []string{"ucs:iam::ucs:policy/AdministratorAccess"},
			"Scope":      "Specified",
			"ProjectID":  d.ProjectID,
		}),
	}, &resp)
	if err != nil {
		return err
	}

	return nil
}

func (d *Driver) grantIamAccess(ctx context.Context, client *api.Client, userName string) error {
	var resp api.IAMAttachPoliciesToUserResponse
	err := client.Do(ctx, api.Request{
		Action: "AttachPoliciesToUser",
		Params: d.actionParams(map[string]any{
			"UserName":   userName,
			"PolicyURNs": []string{"ucs:iam::ucs:policy/IAMFullAccess"},
			"Scope":      "Unspecified",
		}),
	}, &resp)
	if err != nil {
		return err
	}

	return nil
}
