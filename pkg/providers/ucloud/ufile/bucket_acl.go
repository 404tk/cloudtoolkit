package ufile

import (
	"context"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// UCloud UFile bucket access types (the canned-ACL-equivalent UFile concept).
// `private` and `public` are the two stable values; `limited` is "限制公开读"
// (trigger-style access via referer/ip rules) and is mapped through but not a
// default expose target because behavior depends on per-bucket trigger rules.
const (
	UFileTypePrivate = "private"
	UFileTypePublic  = "public"
	UFileTypeLimited = "limited"
)

// AuditBucketACL enumerates buckets and returns the canned access type for
// each. UCloud reports the Type via the same DescribeBucket action used by
// cloudlist, so the audit path is a single JSON-RPC roundtrip per page.
func (d *Driver) AuditBucketACL(ctx context.Context, bucket string) ([]schema.BucketACLEntry, error) {
	bucket = strings.TrimSpace(bucket)
	out := make([]schema.BucketACLEntry, 0)
	offset := 0
	for {
		params := map[string]any{
			"Limit":  pageSize,
			"Offset": offset,
		}
		if bucket != "" {
			params["BucketName"] = bucket
		}
		if region := strings.TrimSpace(d.Region); region != "" && !strings.EqualFold(region, "all") {
			params["Region"] = region
		}
		var resp api.DescribeBucketResponse
		if err := d.client().Do(ctx, api.Request{
			Action: "DescribeBucket",
			Params: params,
		}, &resp); err != nil {
			return out, err
		}
		for _, item := range resp.DataSet {
			name := strings.TrimSpace(item.BucketName)
			if name == "" {
				continue
			}
			if bucket != "" && name != bucket {
				continue
			}
			out = append(out, schema.BucketACLEntry{
				Container: name,
				Level:     normalizeUFileType(item.Type),
			})
		}
		if len(resp.DataSet) == 0 || len(resp.DataSet) < pageSize {
			break
		}
		offset += len(resp.DataSet)
	}
	if bucket != "" && len(out) == 0 {
		return out, fmt.Errorf("ucloud ufile: bucket %q not found", bucket)
	}
	return out, nil
}

// ExposeBucket flips bucket to `public` (or the supplied level if it is a
// recognised UFile type). UCloud `UpdateBucket` only takes BucketName + Type.
func (d *Driver) ExposeBucket(ctx context.Context, bucket, level string) (string, error) {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return "", fmt.Errorf("ucloud ufile: empty bucket")
	}
	target := normalizeUFileType(level)
	if target == "" || target == UFileTypePrivate {
		target = UFileTypePublic
	}
	var resp api.UpdateBucketResponse
	err := d.client().Do(ctx, api.Request{
		Action: "UpdateBucket",
		Params: map[string]any{
			"BucketName": bucket,
			"Type":       target,
		},
	}, &resp)
	if err != nil {
		return "", err
	}
	return target, nil
}

// UnexposeBucket reverts bucket to `private`.
func (d *Driver) UnexposeBucket(ctx context.Context, bucket string) error {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return fmt.Errorf("ucloud ufile: empty bucket")
	}
	var resp api.UpdateBucketResponse
	return d.client().Do(ctx, api.Request{
		Action: "UpdateBucket",
		Params: map[string]any{
			"BucketName": bucket,
			"Type":       UFileTypePrivate,
		},
	}, &resp)
}

// normalizeUFileType maps friendly aliases to the canonical UFile type
// values. Unknown values are passed through lower-cased so the audit table
// surfaces whatever the API returned, even if it's a future-added type.
func normalizeUFileType(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", "private":
		return UFileTypePrivate
	case "public", "public-read", "publicread", "blob", "container":
		return UFileTypePublic
	case "limited", "limited-read", "trigger":
		return UFileTypeLimited
	}
	return strings.ToLower(strings.TrimSpace(level))
}
