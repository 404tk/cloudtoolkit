package iam

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// ListAccessKeys enumerates the API keys belonging to a UCloud IAM sub user.
// The action name follows the same `ListXForUser` family used by UCloud's
// other IAM enumeration RPCs (`ListPoliciesForUser`); verify against the
// upstream SDK before relying on this in production.
func (d *Driver) ListAccessKeys(ctx context.Context, userName string) ([]schema.IAMCredential, error) {
	userName = strings.TrimSpace(userName)
	if userName == "" {
		return nil, fmt.Errorf("ucloud iam: principal (user name) required for list")
	}
	client := d.client()
	out := make([]schema.IAMCredential, 0)
	for offset := 0; ; offset += listUsersPageSize {
		var resp api.IAMListUserApiKeysResponse
		err := client.Do(ctx, api.Request{
			Action: "ListUserApiKeys",
			Params: map[string]any{
				"UserName": userName,
				"Limit":    strconv.Itoa(listUsersPageSize),
				"Offset":   strconv.Itoa(offset),
			},
			Idempotent: true,
		}, &resp)
		if err != nil {
			return nil, err
		}
		for _, k := range resp.ApiKeys {
			out = append(out, schema.IAMCredential{
				CredentialID:   k.AccessKeyID,
				CredentialType: k.Status,
				ValidAfter:     k.CreatedAt,
			})
		}
		if len(resp.ApiKeys) == 0 || len(resp.ApiKeys) < listUsersPageSize ||
			(resp.TotalCount > 0 && offset+len(resp.ApiKeys) >= resp.TotalCount) {
			break
		}
	}
	return out, nil
}

// CreateAccessKey provisions a new API key for the given UCloud IAM user.
// The secret is returned exactly once at creation time.
func (d *Driver) CreateAccessKey(ctx context.Context, userName string) (schema.IAMCredential, string, error) {
	userName = strings.TrimSpace(userName)
	if userName == "" {
		return schema.IAMCredential{}, "", fmt.Errorf("ucloud iam: principal (user name) required for create")
	}
	var resp api.IAMCreateUserApiKeyResponse
	err := d.client().Do(ctx, api.Request{
		Action: "CreateUserApiKey",
		Params: map[string]any{
			"UserName": userName,
		},
	}, &resp)
	if err != nil {
		return schema.IAMCredential{}, "", err
	}
	return schema.IAMCredential{
		CredentialID:   resp.AccessKeyID,
		CredentialType: resp.Status,
		ValidAfter:     resp.CreatedAt,
	}, resp.AccessKeySecret, nil
}

// DeleteAccessKey revokes the API key identified by accessKeyID for userName.
func (d *Driver) DeleteAccessKey(ctx context.Context, userName, accessKeyID string) error {
	userName = strings.TrimSpace(userName)
	accessKeyID = strings.TrimSpace(accessKeyID)
	if userName == "" || accessKeyID == "" {
		return fmt.Errorf("ucloud iam: principal and credential id required for delete")
	}
	var resp api.IAMDeleteUserApiKeyResponse
	return d.client().Do(ctx, api.Request{
		Action: "DeleteUserApiKey",
		Params: map[string]any{
			"UserName":    userName,
			"AccessKeyID": accessKeyID,
		},
	}, &resp)
}
