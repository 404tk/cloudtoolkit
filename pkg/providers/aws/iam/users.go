package iam

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

var errNilAPIClient = errors.New("aws iam: nil api client")

type Driver struct {
	Client        *api.Client
	Region        string
	DefaultRegion string
	Username      string
	Password      string
}

func (d *Driver) ListUsers(ctx context.Context) ([]schema.User, error) {
	list := []schema.User{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List IAM users ...")
	}

	client, err := d.requireClient()
	if err != nil {
		return list, err
	}
	region := d.requestRegion()

	users, err := paginate.Fetch[api.IAMUser, string](ctx, func(ctx context.Context, marker string) (paginate.Page[api.IAMUser, string], error) {
		resp, err := client.ListUsers(ctx, region, marker)
		if err != nil {
			return paginate.Page[api.IAMUser, string]{}, err
		}
		return paginate.Page[api.IAMUser, string]{
			Items: resp.Users,
			Next:  resp.Marker,
			Done:  !resp.IsTruncated || strings.TrimSpace(resp.Marker) == "",
		}, nil
	})
	if err != nil {
		logger.Error("List users failed.")
		return list, err
	}

	for _, user := range users {
		mapped := schema.User{
			UserName:   user.UserName,
			UserId:     user.UserID,
			CreateTime: formatRFC3339(user.CreateDate),
		}
		if user.PasswordLastUsed != nil {
			mapped.LastLogin = user.PasswordLastUsed.Format(time.RFC3339)
			mapped.EnableLogin = true
		} else {
			if _, err := client.GetLoginProfile(ctx, region, mapped.UserName); err == nil {
				mapped.EnableLogin = true
			}
		}
		mapped.Policies = listAttachedUserPolicies(ctx, client, region, mapped.UserName)
		list = append(list, mapped)
	}

	return list, nil
}

func (d *Driver) requireClient() (*api.Client, error) {
	if d.Client == nil {
		return nil, errNilAPIClient
	}
	return d.Client, nil
}

func formatRFC3339(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

func (d *Driver) requestRegion() string {
	region := strings.TrimSpace(d.Region)
	if region == "" || region == "all" {
		region = strings.TrimSpace(d.DefaultRegion)
	}
	if region == "" {
		return "us-east-1"
	}
	return region
}
