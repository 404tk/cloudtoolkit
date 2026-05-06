package iam

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// ListAccessKeys enumerates the permanent access keys of userName via the
// IAM v3.0 OS-CREDENTIAL endpoint. An empty userName resolves to the calling
// user via the AK that established the session.
func (d *Driver) ListAccessKeys(ctx context.Context, userName string) ([]schema.IAMCredential, error) {
	region, err := d.requestRegion()
	if err != nil {
		return nil, err
	}
	userID, err := d.resolvePrincipalID(ctx, userName)
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("user_id", userID)
	var resp api.ListPermanentAccessKeysResponse
	if err := d.client().DoJSON(ctx, api.Request{
		Service:    "iam",
		Region:     region,
		Intl:       d.Cred.Intl,
		Method:     http.MethodGet,
		Path:       "/v3.0/OS-CREDENTIAL/credentials",
		Query:      query,
		Idempotent: true,
	}, &resp); err != nil {
		return nil, err
	}
	out := make([]schema.IAMCredential, 0, len(resp.Credentials))
	for _, c := range resp.Credentials {
		out = append(out, schema.IAMCredential{
			CredentialID:   c.Access,
			CredentialType: c.Status,
			ValidAfter:     c.CreateTime,
		})
	}
	return out, nil
}

// CreateAccessKey provisions a new permanent access key for userName. The
// secret is returned only once at creation time.
func (d *Driver) CreateAccessKey(ctx context.Context, userName string) (schema.IAMCredential, string, error) {
	userName = strings.TrimSpace(userName)
	if userName == "" {
		return schema.IAMCredential{}, "", fmt.Errorf("huawei iam: principal (user name) required for create")
	}
	region, err := d.requestRegion()
	if err != nil {
		return schema.IAMCredential{}, "", err
	}
	userID, err := d.lookupUserID(ctx, userName)
	if err != nil {
		return schema.IAMCredential{}, "", err
	}
	body, err := json.Marshal(api.CreatePermanentAccessKeyRequest{
		Credential: api.CreatePermanentAccessKeyOption{
			UserID:      userID,
			Description: "ctk validation key",
		},
	})
	if err != nil {
		return schema.IAMCredential{}, "", err
	}
	var resp api.CreatePermanentAccessKeyResponse
	if err := d.client().DoJSON(ctx, api.Request{
		Service: "iam",
		Region:  region,
		Intl:    d.Cred.Intl,
		Method:  http.MethodPost,
		Path:    "/v3.0/OS-CREDENTIAL/credentials",
		Body:    body,
	}, &resp); err != nil {
		return schema.IAMCredential{}, "", err
	}
	return schema.IAMCredential{
		CredentialID:   resp.Credential.Access,
		CredentialType: resp.Credential.Status,
		ValidAfter:     resp.Credential.CreateTime,
	}, resp.Credential.Secret, nil
}

// DeleteAccessKey revokes the permanent access key identified by accessKeyID.
// Huawei's DELETE endpoint takes the access key directly, so userName is only
// retained for parity with other providers.
func (d *Driver) DeleteAccessKey(ctx context.Context, _ string, accessKeyID string) error {
	accessKeyID = strings.TrimSpace(accessKeyID)
	if accessKeyID == "" {
		return fmt.Errorf("huawei iam: credential id required for delete")
	}
	region, err := d.requestRegion()
	if err != nil {
		return err
	}
	return d.client().DoJSON(ctx, api.Request{
		Service: "iam",
		Region:  region,
		Intl:    d.Cred.Intl,
		Method:  http.MethodDelete,
		Path:    fmt.Sprintf("/v3.0/OS-CREDENTIAL/credentials/%s", accessKeyID),
	}, nil)
}

func (d *Driver) resolvePrincipalID(ctx context.Context, userName string) (string, error) {
	userName = strings.TrimSpace(userName)
	if userName == "" {
		return d.getUserID(ctx)
	}
	return d.lookupUserID(ctx, userName)
}
