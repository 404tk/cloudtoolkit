package iam

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	ucloudauth "github.com/404tk/cloudtoolkit/pkg/providers/ucloud/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const listUsersPageSize = 100

type Driver struct {
	Credential ucloudauth.Credential
	Client     *api.Client
	ProjectID  string
	UserName   string
	Password   string
}

func (d *Driver) client() *api.Client {
	if d.Client != nil {
		return d.Client
	}
	return api.NewClient(d.Credential)
}

func (d *Driver) ListUsers(ctx context.Context) ([]schema.User, error) {
	list := []schema.User{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List UCloud IAM users ...")
	}

	client := d.client()
	for offset := 0; ; offset += listUsersPageSize {
		var resp api.IAMListUsersResponse
		err := client.Do(ctx, api.Request{
			Action: "ListUsers",
			Params: map[string]any{
				"Limit":  strconv.Itoa(listUsersPageSize),
				"Offset": strconv.Itoa(offset),
			},
			Idempotent: true,
		}, &resp)
		if err != nil {
			logger.Error("List users failed.")
			return list, err
		}

		for _, user := range resp.Users {
			list = append(list, schema.User{
				UserName:    user.DisplayName,
				UserId:      user.UserName,
				EnableLogin: strings.EqualFold(strings.TrimSpace(user.Status), "Active"),
				CreateTime:  formatUnix(user.CreatedAt),
			})
		}

		if len(resp.Users) == 0 || len(resp.Users) < listUsersPageSize || (resp.TotalCount > 0 && offset+len(resp.Users) >= resp.TotalCount) {
			break
		}

		select {
		case <-ctx.Done():
			return list, nil
		default:
		}
	}

	return list, nil
}

func (d *Driver) actionParams(params map[string]any) map[string]any {
	out := make(map[string]any, len(params)+2)
	for key, value := range params {
		out[key] = value
	}

	return out
}

func formatUnix(value int64) string {
	if value <= 0 {
		return ""
	}
	return time.Unix(value, 0).UTC().Format(time.RFC3339)
}
