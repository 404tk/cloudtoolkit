package iam

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// ListAccessKeys enumerates the access keys attached to a JDCloud sub user
// via the pattern-inferred `:describeAccessKeys` action under /subUser. The
// path mirrors the existing `:attachSubUserPolicy` / `:detachSubUserPolicy`
// shape; verify against upstream SDK if behaviour deviates.
func (d *Driver) ListAccessKeys(ctx context.Context, userName string) ([]schema.IAMCredential, error) {
	if d.Client == nil {
		return nil, fmt.Errorf("jdcloud iam: nil api client")
	}
	userName = strings.TrimSpace(userName)
	if userName == "" {
		return nil, fmt.Errorf("jdcloud iam: principal (sub user name) required for list")
	}
	var resp api.DescribeAccessKeysResponse
	if err := d.Client.DoJSON(ctx, api.Request{
		Service: "iam",
		Region:  "",
		Method:  http.MethodGet,
		Version: "v1",
		Path:    "/subUser/" + userName + ":describeAccessKeys",
	}, &resp); err != nil {
		return nil, err
	}
	out := make([]schema.IAMCredential, 0, len(resp.Result.AccessKeys))
	for _, k := range resp.Result.AccessKeys {
		out = append(out, schema.IAMCredential{
			CredentialID:   k.AccessKey,
			CredentialType: k.Status,
			ValidAfter:     k.CreateTime,
		})
	}
	return out, nil
}

// CreateAccessKey provisions a new access key pair for the sub user via
// `:createAccessKey`. Returns the public AccessKey and the one-shot Secret.
func (d *Driver) CreateAccessKey(ctx context.Context, userName string) (schema.IAMCredential, string, error) {
	if d.Client == nil {
		return schema.IAMCredential{}, "", fmt.Errorf("jdcloud iam: nil api client")
	}
	userName = strings.TrimSpace(userName)
	if userName == "" {
		return schema.IAMCredential{}, "", fmt.Errorf("jdcloud iam: principal (sub user name) required for create")
	}
	body, err := json.Marshal(api.CreateAccessKeyRequest{SubUser: userName})
	if err != nil {
		return schema.IAMCredential{}, "", err
	}
	var resp api.CreateAccessKeyResponse
	if err := d.Client.DoJSON(ctx, api.Request{
		Service: "iam",
		Region:  "",
		Method:  http.MethodPost,
		Version: "v1",
		Path:    "/subUser/" + userName + ":createAccessKey",
		Body:    body,
	}, &resp); err != nil {
		return schema.IAMCredential{}, "", err
	}
	return schema.IAMCredential{
		CredentialID:   resp.Result.AccessKey.AccessKey,
		CredentialType: resp.Result.AccessKey.Status,
		ValidAfter:     resp.Result.AccessKey.CreateTime,
	}, resp.Result.AccessKey.SecretKey, nil
}

// DeleteAccessKey revokes an access key for the sub user via
// `:deleteAccessKey`. The accessKey is passed via query parameter to match the
// `:detachSubUserPolicy` DELETE convention.
func (d *Driver) DeleteAccessKey(ctx context.Context, userName, accessKeyID string) error {
	if d.Client == nil {
		return fmt.Errorf("jdcloud iam: nil api client")
	}
	userName = strings.TrimSpace(userName)
	accessKeyID = strings.TrimSpace(accessKeyID)
	if userName == "" || accessKeyID == "" {
		return fmt.Errorf("jdcloud iam: principal and credential id required for delete")
	}
	query := url.Values{}
	query.Set("accessKey", accessKeyID)
	var resp api.DeleteAccessKeyResponse
	return d.Client.DoJSON(ctx, api.Request{
		Service: "iam",
		Region:  "",
		Method:  http.MethodDelete,
		Version: "v1",
		Path:    "/subUser/" + userName + ":deleteAccessKey",
		Query:   query,
	}, &resp)
}
