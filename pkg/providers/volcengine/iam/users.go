package iam

import (
	"context"
	"fmt"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Client   *api.Client
	Region   string
	UserName string
	Password string
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
	var offset int32 = 0
	for {
		resp, err := client.ListUsers(ctx, region, 100, offset)
		if err != nil {
			logger.Error("List users failed.")
			return list, err
		}

		for _, user := range resp.Result.UserMetadata {
			_user := schema.User{
				UserName: user.UserName,
				UserId:   fmt.Sprint(user.AccountID),
			}
			date, _ := time.Parse("20060102T150405Z", user.CreateDate)
			_user.CreateTime = date.String()

			lresp, err := client.GetLoginProfile(ctx, region, user.UserName)
			if err == nil {
				_user.EnableLogin = lresp.Result.LoginProfile.LoginAllowed
				ldate := lresp.Result.LoginProfile.LastLoginDate
				if _user.EnableLogin && ldate != "" && ldate != "19700101T000000Z" {
					lastLoginDate, _ := time.Parse("20060102T150405Z", ldate)
					_user.LastLogin = lastLoginDate.String()
				}

			}

			list = append(list, _user)
			select {
			case <-ctx.Done():
				return list, nil
			default:
				continue
			}
		}
		if len(resp.Result.UserMetadata) < 100 || resp.Result.Total == offset {
			break
		}
		offset += 100
	}
	return list, nil
}
