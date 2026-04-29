package iam

import (
	"context"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

func (d *Driver) DelUser() (schema.IAMResult, error) {
	ctx := context.Background()
	client := api.NewClient(d.Credential)
	msg, err := d.deleteUser(ctx, client, d.UserName)
	if err != nil {
		return schema.IAMResult{}, fmt.Errorf("delete user %s failed: %w", d.UserName, err)
	}

	return schema.IAMResult{
		Username: d.UserName,
		Message:  msg,
	}, nil
}

func (d *Driver) deleteUser(ctx context.Context, client *api.Client, userName string) (string, error) {
	var resp api.IAMDeleteUserResponse
	err := client.Do(ctx, api.Request{
		Action: "DeleteUser",
		Params: d.actionParams(map[string]any{
			"UserName": userName,
		}),
	}, &resp)
	if err != nil {
		return "", err
	}

	message := strings.TrimSpace(resp.Message)
	if message == "" || strings.EqualFold(message, "success") {
		message = "User deleted successfully"
	}

	return message, nil
}
