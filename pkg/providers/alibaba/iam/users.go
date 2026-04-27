package iam

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Cred          aliauth.Credential
	Region        string
	UserName      string
	Password      string
	RoleName      string
	AccountId     string
	clientOptions []api.Option
}

func (d *Driver) newClient() *api.Client {
	return api.NewClient(d.Cred, d.clientOptions...)
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
		logger.Info("List RAM users ...")
	}
	client := d.newClient()
	region := api.NormalizeRegion(d.Region)
	policyInfos = make(map[string]string)

	users, err := paginate.Fetch(ctx, func(ctx context.Context, marker string) (paginate.Page[api.RAMUser, string], error) {
		response, err := client.ListRAMUsers(ctx, region, marker, 100)
		if err != nil {
			logger.Error("List users failed.")
			return paginate.Page[api.RAMUser, string]{}, err
		}
		return paginate.Page[api.RAMUser, string]{
			Items: response.Users.User,
			Next:  response.Marker,
			Done:  !response.IsTruncated,
		}, nil
	})
	if err != nil {
		return list, err
	}

	for _, user := range users {
		_user := schema.User{
			UserName:   user.UserName,
			UserId:     user.UserID,
			CreateTime: formatRAMTime(user.CreateDate),
		}

		enableLogin, lastLogin := lookupLoginState(ctx, client, region, user.UserName)
		_user.EnableLogin = enableLogin
		_user.LastLogin = lastLogin

		if env.From(ctx).ListPolicies {
			_user.Policies = listPoliciesForUser(ctx, client, region, _user.UserName)
		}

		list = append(list, _user)
		select {
		case <-ctx.Done():
			return list, nil
		default:
		}
	}

	return list, nil
}

func lookupLoginState(ctx context.Context, client *api.Client, region, userName string) (bool, string) {
	if _, err := client.GetRAMLoginProfile(ctx, region, userName); err != nil {
		if isMissingLoginProfileError(err) {
			return false, ""
		}
		logger.Error("Get login profile failed:", err)
		return false, ""
	}

	response, err := client.GetRAMUser(ctx, region, userName)
	if err != nil {
		logger.Error("Get user failed:", err)
		return true, ""
	}
	return true, formatRAMTime(response.User.LastLoginDate)
}

func formatRAMTime(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	date, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return date.String()
}

func isMissingLoginProfileError(err error) bool {
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		if strings.EqualFold(apiErr.Code, "EntityNotExist.LoginProfile") {
			return true
		}
		return strings.Contains(strings.ToLower(apiErr.Message), "login policy not exists")
	}
	return strings.Contains(strings.ToLower(err.Error()), "login policy not exists")
}

func isEntityNotExistError(err error) bool {
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		return strings.HasPrefix(strings.TrimSpace(apiErr.Code), "EntityNotExist.")
	}
	return strings.Contains(strings.ToLower(err.Error()), "entitynotexist")
}
