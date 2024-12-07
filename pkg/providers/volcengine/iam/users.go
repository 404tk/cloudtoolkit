package iam

import (
	"context"
	"fmt"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/volcengine/volcengine-go-sdk/service/iam"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
)

type Driver struct {
	Conf *volcengine.Config
}

func (d *Driver) NewClient() *iam.IAM {
	sess, _ := session.NewSession(d.Conf.WithRegion("cn-beijing"))
	return iam.New(sess)
}

func (d *Driver) ListUsers(ctx context.Context) ([]schema.User, error) {
	list := []schema.User{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List IAM users ...")
	}
	svc := d.NewClient()
	var offset int32 = 0
	//policy_infos = make(map[string]string)
	for {
		listUsersInput := &iam.ListUsersInput{
			Limit:  volcengine.Int32(100),
			Offset: volcengine.Int32(offset),
		}
		resp, err := svc.ListUsers(listUsersInput)
		if err != nil {
			logger.Error("List users failed.")
			return list, err
		}

		for _, user := range resp.UserMetadata {
			_user := schema.User{
				UserName: volcengine.StringValue(user.UserName),
				UserId:   fmt.Sprint(*user.AccountId),
			}
			date, _ := time.Parse("20060102T150405Z", volcengine.StringValue(user.CreateDate))
			_user.CreateTime = date.String()

			input := &iam.GetLoginProfileInput{UserName: user.UserName}
			lresp, err := svc.GetLoginProfile(input)
			if err == nil {
				_user.EnableLogin = true
				ldate := volcengine.StringValue(lresp.LoginProfile.LastLoginDate)
				if ldate != "" && ldate != "19700101T000000Z" {
					lastLoginDate, _ := time.Parse("20060102T150405Z", ldate)
					_user.LastLogin = lastLoginDate.String()
				}

			}

			if utils.ListPolicies {
				// _user.Policies = listPoliciesForUser(client, _user.UserName)
			}

			list = append(list, _user)
			select {
			case <-ctx.Done():
				return list, nil
			default:
				continue
			}
		}
		if len(resp.UserMetadata) < 100 || *resp.Total == offset {
			break
		}
		offset += 100
	}
	return list, nil
}
