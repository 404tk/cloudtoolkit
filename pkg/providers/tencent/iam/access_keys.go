package iam

import (
	"context"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// ListAccessKeys returns the CAM access keys belonging to userName. Tencent
// CAM keys are owned by a Uin, so when userName is non-empty the driver
// resolves it via GetUser; an empty userName lists keys for the calling
// principal.
func (d *Driver) ListAccessKeys(ctx context.Context, userName string) ([]schema.IAMCredential, error) {
	client := d.newClient()
	uin, err := d.resolveTargetUin(ctx, userName)
	if err != nil {
		return nil, err
	}
	resp, err := client.ListAccessKeys(ctx, uin)
	if err != nil {
		return nil, err
	}
	out := make([]schema.IAMCredential, 0, len(resp.Response.AccessKeys))
	for _, k := range resp.Response.AccessKeys {
		out = append(out, schema.IAMCredential{
			CredentialID:   derefString(k.AccessKeyID),
			CredentialType: derefString(k.Status),
			ValidAfter:     derefString(k.CreateTime),
		})
	}
	return out, nil
}

// CreateAccessKey mints a fresh CAM access key pair. Tencent returns the
// secret only on creation; capture it from the second return value.
func (d *Driver) CreateAccessKey(ctx context.Context, userName string) (schema.IAMCredential, string, error) {
	userName = strings.TrimSpace(userName)
	if userName == "" {
		return schema.IAMCredential{}, "", fmt.Errorf("tencent iam: principal (CAM user name) required for create")
	}
	client := d.newClient()
	uin, err := d.resolveTargetUin(ctx, userName)
	if err != nil {
		return schema.IAMCredential{}, "", err
	}
	resp, err := client.CreateAccessKey(ctx, uin)
	if err != nil {
		return schema.IAMCredential{}, "", err
	}
	return schema.IAMCredential{
		CredentialID:   derefString(resp.Response.AccessKey.AccessKeyID),
		CredentialType: derefString(resp.Response.AccessKey.Status),
		ValidAfter:     derefString(resp.Response.AccessKey.CreateTime),
	}, derefString(resp.Response.AccessKey.SecretAccessKey), nil
}

// DeleteAccessKey revokes a CAM access key by ID for userName.
func (d *Driver) DeleteAccessKey(ctx context.Context, userName, accessKeyID string) error {
	accessKeyID = strings.TrimSpace(accessKeyID)
	if accessKeyID == "" {
		return fmt.Errorf("tencent iam: credential id required for delete")
	}
	client := d.newClient()
	uin, err := d.resolveTargetUin(ctx, userName)
	if err != nil {
		return err
	}
	_, err = client.DeleteAccessKey(ctx, uin, accessKeyID)
	return err
}

func (d *Driver) resolveTargetUin(ctx context.Context, userName string) (uint64, error) {
	userName = strings.TrimSpace(userName)
	if userName == "" {
		return 0, nil
	}
	client := d.newClient()
	resp, err := client.GetUser(ctx, userName)
	if err != nil {
		return 0, err
	}
	uin := derefUint64(resp.Response.Uin)
	if uin == 0 {
		return 0, fmt.Errorf("tencent iam: cannot resolve uin for user %s", userName)
	}
	return uin, nil
}
