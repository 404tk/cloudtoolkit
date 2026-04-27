package iam

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Credential    auth.Credential
	UserName      string
	Password      string
	RoleName      string
	Uin           string
	clientOptions []api.Option
}

func (d *Driver) newClient() *api.Client {
	return api.NewClient(d.Credential, d.clientOptions...)
}

func (d *Driver) SetClientOptions(opts ...api.Option) {
	d.clientOptions = append([]api.Option(nil), opts...)
}

func (d *Driver) ListUsers(ctx context.Context) ([]schema.User, error) {
	list := []schema.User{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List CAM users ...")
	}
	client := d.newClient()
	listUsersResponse, err := client.ListUsers(ctx)
	if err != nil {
		logger.Error("List users failed.")
		return list, err
	}
	policy_infos = make(map[string]string)
	for _, user := range listUsersResponse.Response.Data {
		_user := schema.User{
			UserName:   derefString(user.Name),
			UserId:     fmt.Sprintf("%v", derefUint64(user.Uin)),
			CreateTime: derefString(user.CreateTime),
		}
		if derefUint64(user.ConsoleLogin) == 1 {
			_user.EnableLogin = true
		}
		if env.From(ctx).ListPolicies {
			_user.Policies = listAttachedUserAllPolicies(ctx, client, derefUint64(user.Uin))
		}

		list = append(list, _user)
	}
	return list, nil
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func derefUint64(v *uint64) uint64 {
	if v == nil {
		return 0
	}
	return *v
}
