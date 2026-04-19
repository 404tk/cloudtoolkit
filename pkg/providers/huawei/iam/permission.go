package iam

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
)

func (d *Driver) GetUserName(ctx context.Context) (string, error) {
	userID, err := d.getUserID(ctx)
	if err != nil {
		return "", err
	}

	region, err := d.requestRegion()
	if err != nil {
		return "", err
	}
	var resp api.ShowUserResponse
	if err := d.client().DoJSON(ctx, api.Request{
		Service:    "iam",
		Region:     region,
		Intl:       d.Cred.Intl,
		Method:     http.MethodGet,
		Path:       fmt.Sprintf("/v3/users/%s", userID),
		Idempotent: true,
	}, &resp); err != nil {
		return "", err
	}
	if resp.User.DomainID != "" {
		d.DomainID = resp.User.DomainID
	}
	return resp.User.Name, nil
}

func (d *Driver) getUserID(ctx context.Context) (string, error) {
	region, err := d.requestRegion()
	if err != nil {
		return "", err
	}

	var resp api.ShowPermanentAccessKeyResponse
	if err := d.client().DoJSON(ctx, api.Request{
		Service:    "iam",
		Region:     region,
		Intl:       d.Cred.Intl,
		Method:     http.MethodGet,
		Path:       fmt.Sprintf("/v3.0/OS-CREDENTIAL/credentials/%s", d.Cred.AK),
		Idempotent: true,
	}, &resp); err != nil {
		return "", err
	}
	if resp.Credential.UserID == "" && resp.ErrorMsg != "" {
		return "", errors.New(resp.ErrorMsg)
	}
	return resp.Credential.UserID, nil
}
