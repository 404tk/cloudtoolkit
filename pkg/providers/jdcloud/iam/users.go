package iam

import (
	"context"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/jdcloud-api/jdcloud-sdk-go/core"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/iam/apis"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/iam/client"
)

type Driver struct {
	Cred  *core.Credential
	Token string
}

func (d *Driver) newClient() *client.IamClient {
	c := client.NewIamClient(d.Cred)
	c.SetLogger(core.NewDummyLogger())
	return c
}

func (d *Driver) ListUsers(ctx context.Context) ([]schema.User, error) {
	list := []schema.User{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List IAM users ...")
	}
	svc := d.newClient()
	req := apis.NewDescribeSubUsersRequest()
	req.AddHeader("x-jdcloud-security-token", d.Token)
	resp, err := svc.DescribeSubUsers(req)
	if err != nil {
		logger.Error("List users failed.")
		return list, err
	}

	for _, user := range resp.Result.SubUsers {
		list = append(list, schema.User{
			UserName:   user.Name,
			UserId:     user.Account,
			CreateTime: user.CreateTime,
		})
	}
	return list, nil
}

func (d *Driver) Validator(user string) bool {
	svc := d.newClient()
	req := apis.NewDescribeSubUserRequest(user)
	resp, err := svc.DescribeSubUser(req)
	if err != nil {
		return false
	}
	if resp.Error.Code == 404 || resp.Result.SubUser.Name != "" {
		return true
	}
	return false
}
