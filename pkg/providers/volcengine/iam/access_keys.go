package iam

import (
	"context"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// ListAccessKeys returns the IAM access keys belonging to userName. Volcengine
// returns the calling user's keys when UserName is omitted.
func (d *Driver) ListAccessKeys(ctx context.Context, userName string) ([]schema.IAMCredential, error) {
	client, err := d.requireClient()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListAccessKeys(ctx, d.requestRegion(), userName)
	if err != nil {
		return nil, err
	}
	out := make([]schema.IAMCredential, 0, len(resp.Result.AccessKeyMetadata))
	for _, k := range resp.Result.AccessKeyMetadata {
		out = append(out, schema.IAMCredential{
			CredentialID:   k.AccessKeyID,
			CredentialType: k.Status,
			ValidAfter:     k.CreateDate,
		})
	}
	return out, nil
}

// CreateAccessKey mints a fresh AccessKey pair for userName.
func (d *Driver) CreateAccessKey(ctx context.Context, userName string) (schema.IAMCredential, string, error) {
	userName = strings.TrimSpace(userName)
	if userName == "" {
		return schema.IAMCredential{}, "", fmt.Errorf("volcengine iam: principal (IAM user name) required for create")
	}
	client, err := d.requireClient()
	if err != nil {
		return schema.IAMCredential{}, "", err
	}
	resp, err := client.CreateAccessKey(ctx, d.requestRegion(), userName)
	if err != nil {
		return schema.IAMCredential{}, "", err
	}
	return schema.IAMCredential{
		CredentialID:   resp.Result.AccessKey.AccessKeyID,
		CredentialType: resp.Result.AccessKey.Status,
		ValidAfter:     resp.Result.AccessKey.CreateDate,
	}, resp.Result.AccessKey.SecretAccessKey, nil
}

// DeleteAccessKey revokes accessKeyID belonging to userName.
func (d *Driver) DeleteAccessKey(ctx context.Context, userName, accessKeyID string) error {
	userName = strings.TrimSpace(userName)
	accessKeyID = strings.TrimSpace(accessKeyID)
	if userName == "" || accessKeyID == "" {
		return fmt.Errorf("volcengine iam: principal and credential id required for delete")
	}
	client, err := d.requireClient()
	if err != nil {
		return err
	}
	_, err = client.DeleteAccessKey(ctx, d.requestRegion(), userName, accessKeyID)
	return err
}
