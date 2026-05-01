package s3

import (
	"context"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// AWS S3 canned ACL values accepted by the `x-amz-acl` header.
const (
	S3ACLPrivate           = "private"
	S3ACLPublicRead        = "public-read"
	S3ACLPublicReadWrite   = "public-read-write"
	S3ACLAuthenticatedRead = "authenticated-read"
	S3ACLAWSExecRead       = "aws-exec-read"
)

// AuditBucketACL enumerates buckets in scope and returns the canned ACL
// summary for each. ACL state on AWS S3 is the union of canned ACL grants
// and a bucket's optional Public Access Block; this view surfaces only the
// canned-grant signal because that is what the bucket-acl-check `audit`
// table is shaped around.
func (d *Driver) AuditBucketACL(ctx context.Context, bucket string) ([]schema.BucketACLEntry, error) {
	client, err := d.requireClient()
	if err != nil {
		return nil, err
	}
	storages, err := d.GetBuckets(ctx)
	if err != nil {
		return nil, err
	}
	bucket = strings.TrimSpace(bucket)
	out := make([]schema.BucketACLEntry, 0, len(storages))
	for _, s := range storages {
		if bucket != "" && s.BucketName != bucket {
			continue
		}
		region := s.Region
		if region == "" {
			region = d.defaultRegion()
		}
		acl, err := client.GetBucketAcl(ctx, region, s.BucketName)
		if err != nil {
			return out, fmt.Errorf("get acl for %s: %w", s.BucketName, err)
		}
		out = append(out, schema.BucketACLEntry{
			Container: s.BucketName,
			Level:     S3CannedACLFromGrants(acl),
		})
	}
	return out, nil
}

// ExposeBucket sets a public canned ACL on bucket. AWS layers Public Access
// Block on top of ACL grants — newer accounts have BPA enabled by default,
// which silently overrides any public canned ACL. This helper deletes the
// bucket-level BPA first (best-effort, errors are non-fatal) so the canned
// ACL change actually surfaces in the next audit.
func (d *Driver) ExposeBucket(ctx context.Context, bucket, level string) (string, error) {
	cannedACL := NormalizeS3ACL(level)
	if cannedACL == "" || cannedACL == S3ACLPrivate {
		cannedACL = S3ACLPublicRead
	}
	region, err := d.bucketRegion(ctx, bucket)
	if err != nil {
		return "", err
	}
	client, err := d.requireClient()
	if err != nil {
		return "", err
	}
	// Best-effort BPA clear. A 404 is the common case (BPA never set); other
	// errors get swallowed because the user may not hold the
	// `s3:PutBucketPublicAccessBlock` permission, but might still want the
	// canned ACL change attempted.
	_ = client.DeletePublicAccessBlock(ctx, region, bucket)
	if err := client.PutBucketAcl(ctx, region, bucket, cannedACL); err != nil {
		return "", err
	}
	return cannedACL, nil
}

// UnexposeBucket reverts bucket to the `private` canned ACL.
func (d *Driver) UnexposeBucket(ctx context.Context, bucket string) error {
	region, err := d.bucketRegion(ctx, bucket)
	if err != nil {
		return err
	}
	client, err := d.requireClient()
	if err != nil {
		return err
	}
	return client.PutBucketAcl(ctx, region, bucket, S3ACLPrivate)
}

func (d *Driver) bucketRegion(ctx context.Context, bucket string) (string, error) {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return "", fmt.Errorf("aws s3: empty bucket")
	}
	storages, err := d.GetBuckets(ctx)
	if err != nil {
		return "", err
	}
	for _, s := range storages {
		if s.BucketName == bucket {
			if s.Region != "" {
				return s.Region, nil
			}
			break
		}
	}
	return d.defaultRegion(), nil
}

// NormalizeS3ACL maps user-friendly aliases to canned S3 ACL values.
func NormalizeS3ACL(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", "private":
		return S3ACLPrivate
	case "public-read", "publicread", "blob", "container":
		return S3ACLPublicRead
	case "public-read-write", "publicreadwrite", "rw", "writable":
		return S3ACLPublicReadWrite
	case "authenticated-read", "authenticatedread":
		return S3ACLAuthenticatedRead
	case "aws-exec-read":
		return S3ACLAWSExecRead
	}
	return strings.ToLower(strings.TrimSpace(level))
}

// S3CannedACLFromGrants collapses a parsed Grant list into the canned-ACL
// label that best represents it. The mapping mirrors how the AWS console
// summarises ACL state: any AllUsers group grant means public.
func S3CannedACLFromGrants(out api.GetBucketAclOutput) string {
	publicRead := false
	publicWrite := false
	for _, grant := range out.Grants {
		uri := strings.ToLower(strings.TrimSpace(grant.GranteeURI))
		if !strings.HasSuffix(uri, "global/allusers") &&
			!strings.HasSuffix(uri, "groups/global/allusers") {
			continue
		}
		switch grant.Permission {
		case "READ":
			publicRead = true
		case "WRITE":
			publicWrite = true
		case "FULL_CONTROL":
			publicRead = true
			publicWrite = true
		}
	}
	switch {
	case publicRead && publicWrite:
		return S3ACLPublicReadWrite
	case publicRead:
		return S3ACLPublicRead
	default:
		return S3ACLPrivate
	}
}
