package iam

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// ListAccessKeys enumerates IAM access keys for userName. AWS requires the user
// name when the caller is not the same principal; an empty userName lets AWS
// fall back to the current user.
func (d *Driver) ListAccessKeys(ctx context.Context, userName string) ([]schema.IAMCredential, error) {
	client, err := d.requireClient()
	if err != nil {
		return nil, err
	}
	region := d.requestRegion()
	keys, err := paginate.Fetch[api.IAMAccessKey, string](ctx, func(ctx context.Context, marker string) (paginate.Page[api.IAMAccessKey, string], error) {
		resp, err := client.ListAccessKeys(ctx, region, userName, marker)
		if err != nil {
			return paginate.Page[api.IAMAccessKey, string]{}, err
		}
		return paginate.Page[api.IAMAccessKey, string]{
			Items: resp.AccessKeys,
			Next:  resp.Marker,
			Done:  !resp.IsTruncated || strings.TrimSpace(resp.Marker) == "",
		}, nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]schema.IAMCredential, 0, len(keys))
	for _, k := range keys {
		out = append(out, schema.IAMCredential{
			CredentialID:   k.AccessKeyID,
			CredentialType: k.Status,
			ValidAfter:     formatRFC3339(k.CreateDate),
		})
	}
	return out, nil
}

// CreateAccessKey mints a new IAM access key for userName. The secret is
// returned once and only once on creation; callers must capture it.
func (d *Driver) CreateAccessKey(ctx context.Context, userName string) (schema.IAMCredential, string, error) {
	userName = strings.TrimSpace(userName)
	if userName == "" {
		return schema.IAMCredential{}, "", fmt.Errorf("aws iam: principal (IAM user name) required for create")
	}
	client, err := d.requireClient()
	if err != nil {
		return schema.IAMCredential{}, "", err
	}
	resp, err := client.CreateAccessKey(ctx, d.requestRegion(), userName)
	if err != nil {
		return schema.IAMCredential{}, "", err
	}
	cred := schema.IAMCredential{
		CredentialID:   resp.AccessKey.AccessKeyID,
		CredentialType: resp.AccessKey.Status,
	}
	if resp.AccessKey.CreateDate != nil {
		cred.ValidAfter = resp.AccessKey.CreateDate.Format(time.RFC3339)
	}
	return cred, resp.AccessKey.SecretAccessKey, nil
}

// DeleteAccessKey revokes an IAM access key by ID.
func (d *Driver) DeleteAccessKey(ctx context.Context, userName, accessKeyID string) error {
	accessKeyID = strings.TrimSpace(accessKeyID)
	if accessKeyID == "" {
		return fmt.Errorf("aws iam: credential id required for delete")
	}
	client, err := d.requireClient()
	if err != nil {
		return err
	}
	return client.DeleteAccessKey(ctx, d.requestRegion(), userName, accessKeyID)
}
