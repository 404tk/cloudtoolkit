package iam

import (
	"context"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// ListAccessKeys returns the RAM AccessKey records belonging to userName.
// An empty userName is allowed and asks RAM to return the keys of the calling
// principal — useful when the caller already proves access via the AK in use.
func (d *Driver) ListAccessKeys(ctx context.Context, userName string) ([]schema.IAMCredential, error) {
	if d == nil {
		return nil, fmt.Errorf("alibaba iam: nil driver")
	}
	client := d.newClient()
	region := api.NormalizeRegion(d.Region)
	resp, err := client.ListRAMAccessKeys(ctx, region, strings.TrimSpace(userName))
	if err != nil {
		return nil, err
	}
	out := make([]schema.IAMCredential, 0, len(resp.AccessKeys.AccessKey))
	for _, k := range resp.AccessKeys.AccessKey {
		out = append(out, schema.IAMCredential{
			CredentialID:   k.AccessKeyID,
			CredentialType: k.Status,
			ValidAfter:     k.CreateDate,
		})
	}
	return out, nil
}

// CreateAccessKey provisions a fresh AccessKey pair for userName.
func (d *Driver) CreateAccessKey(ctx context.Context, userName string) (schema.IAMCredential, string, error) {
	if d == nil {
		return schema.IAMCredential{}, "", fmt.Errorf("alibaba iam: nil driver")
	}
	userName = strings.TrimSpace(userName)
	if userName == "" {
		return schema.IAMCredential{}, "", fmt.Errorf("alibaba iam: principal (RAM user name) required for create")
	}
	client := d.newClient()
	region := api.NormalizeRegion(d.Region)
	resp, err := client.CreateRAMAccessKey(ctx, region, userName)
	if err != nil {
		return schema.IAMCredential{}, "", err
	}
	return schema.IAMCredential{
		CredentialID:   resp.AccessKey.AccessKeyID,
		CredentialType: resp.AccessKey.Status,
		ValidAfter:     resp.AccessKey.CreateDate,
	}, resp.AccessKey.AccessKeySecret, nil
}

// DeleteAccessKey revokes the AccessKey identified by accessKeyID for userName.
func (d *Driver) DeleteAccessKey(ctx context.Context, userName, accessKeyID string) error {
	if d == nil {
		return fmt.Errorf("alibaba iam: nil driver")
	}
	accessKeyID = strings.TrimSpace(accessKeyID)
	if accessKeyID == "" {
		return fmt.Errorf("alibaba iam: credential id required for delete")
	}
	client := d.newClient()
	region := api.NormalizeRegion(d.Region)
	_, err := client.DeleteRAMAccessKey(ctx, region, strings.TrimSpace(userName), accessKeyID)
	return err
}
