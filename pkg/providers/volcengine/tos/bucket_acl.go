package tos

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// TOS canned ACL values. PUT bucket ACL accepts these via the `x-tos-acl`
// header; GET bucket ACL replies with a Grants list that we collapse back to
// one of these canonical values for the bucket-acl-check `audit` view.
const (
	TOSACLPrivate                = "private"
	TOSACLPublicRead             = "public-read"
	TOSACLPublicReadWrite        = "public-read-write"
	TOSACLAuthenticatedRead      = "authenticated-read"
	TOSACLBucketOwnerRead        = "bucket-owner-read"
	TOSACLBucketOwnerFullControl = "bucket-owner-full-control"
)

// GetBucketACLOutput captures the JSON returned by `GET /?acl`. Only the
// fields needed to derive a canonical ACL string are kept.
type GetBucketACLOutput struct {
	Owner struct {
		ID string `json:"ID"`
	} `json:"Owner"`
	Grants []struct {
		Grantee struct {
			Type string `json:"Type"`
			URI  string `json:"Canned"`
			ID   string `json:"ID"`
		} `json:"Grantee"`
		Permission string `json:"Permission"`
	} `json:"Grants"`
}

// GetBucketACL returns the canned ACL value for bucket. The TOS data plane
// returns a permission grant list rather than a direct canned value, so the
// helper collapses Grants back into one of the TOSACL* constants.
func (c *Client) GetBucketACL(ctx context.Context, bucket, region string) (string, error) {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return "", fmt.Errorf("volcengine tos: empty bucket")
	}
	query := url.Values{}
	query.Set("acl", "")
	var out GetBucketACLOutput
	err := c.doJSON(ctx, request{
		Method: http.MethodGet,
		Host:   bucketHost(bucket, region),
		Path:   "/",
		Query:  query,
	}, &out)
	if err != nil {
		return "", err
	}
	return collapseTOSGrants(out), nil
}

// PutBucketACL sets the canned ACL on bucket via the `x-tos-acl` header.
func (c *Client) PutBucketACL(ctx context.Context, bucket, region, acl string) error {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return fmt.Errorf("volcengine tos: empty bucket")
	}
	acl = strings.TrimSpace(acl)
	if acl == "" {
		return fmt.Errorf("volcengine tos: empty acl")
	}
	query := url.Values{}
	query.Set("acl", "")
	headers := http.Header{}
	headers.Set("x-tos-acl", acl)
	return c.doJSON(ctx, request{
		Method:  http.MethodPut,
		Host:    bucketHost(bucket, region),
		Path:    "/",
		Query:   query,
		Headers: headers,
	}, nil)
}

// AuditBucketACL enumerates buckets in scope and returns their canned ACL.
func (d *Driver) AuditBucketACL(ctx context.Context, bucket string) ([]schema.BucketACLEntry, error) {
	client, err := d.NewClient()
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
			region = d.Region
		}
		acl, err := client.GetBucketACL(ctx, s.BucketName, region)
		if err != nil {
			return out, fmt.Errorf("get acl for %s: %w", s.BucketName, err)
		}
		out = append(out, schema.BucketACLEntry{
			Container: s.BucketName,
			Level:     acl,
		})
	}
	return out, nil
}

// ExposeBucket sets the bucket public-readable (defaults to public-read).
func (d *Driver) ExposeBucket(ctx context.Context, bucket, level string) (string, error) {
	acl := NormalizeTOSACL(level)
	if acl == "" || acl == TOSACLPrivate {
		acl = TOSACLPublicRead
	}
	region, err := d.bucketRegion(ctx, bucket)
	if err != nil {
		return "", err
	}
	client, err := d.NewClient()
	if err != nil {
		return "", err
	}
	if err := client.PutBucketACL(ctx, bucket, region, acl); err != nil {
		return "", err
	}
	return acl, nil
}

// UnexposeBucket reverts the bucket to private.
func (d *Driver) UnexposeBucket(ctx context.Context, bucket string) error {
	region, err := d.bucketRegion(ctx, bucket)
	if err != nil {
		return err
	}
	client, err := d.NewClient()
	if err != nil {
		return err
	}
	return client.PutBucketACL(ctx, bucket, region, TOSACLPrivate)
}

func (d *Driver) bucketRegion(ctx context.Context, bucket string) (string, error) {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return "", fmt.Errorf("volcengine tos: empty bucket")
	}
	if d.Region != "" && d.Region != "all" {
		return d.Region, nil
	}
	storages, err := d.GetBuckets(ctx)
	if err != nil {
		return "", err
	}
	for _, s := range storages {
		if s.BucketName == bucket {
			return s.Region, nil
		}
	}
	return "", fmt.Errorf("volcengine tos: region for bucket %q not found", bucket)
}

// NormalizeTOSACL maps user-friendly aliases to canned TOS ACL values.
func NormalizeTOSACL(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", "private":
		return TOSACLPrivate
	case "public-read", "publicread", "blob", "container":
		return TOSACLPublicRead
	case "public-read-write", "publicreadwrite", "rw", "writable":
		return TOSACLPublicReadWrite
	case "authenticated-read", "authenticatedread":
		return TOSACLAuthenticatedRead
	case "bucket-owner-read":
		return TOSACLBucketOwnerRead
	case "bucket-owner-full-control":
		return TOSACLBucketOwnerFullControl
	}
	return strings.ToLower(strings.TrimSpace(level))
}

// collapseTOSGrants reduces a list of permission grants back into a single
// canned ACL value. The result is best-effort: any grant referencing the
// `AllUsers` group is treated as the corresponding public-* value.
func collapseTOSGrants(out GetBucketACLOutput) string {
	hasPublicRead := false
	hasPublicWrite := false
	for _, grant := range out.Grants {
		uri := strings.ToLower(strings.TrimSpace(grant.Grantee.URI))
		if grant.Grantee.Type != "Group" && uri == "" {
			continue
		}
		if uri != "allusers" && uri != "anonymous" {
			continue
		}
		switch strings.ToUpper(strings.TrimSpace(grant.Permission)) {
		case "READ":
			hasPublicRead = true
		case "WRITE":
			hasPublicWrite = true
		case "FULL_CONTROL":
			hasPublicRead = true
			hasPublicWrite = true
		}
	}
	switch {
	case hasPublicRead && hasPublicWrite:
		return TOSACLPublicReadWrite
	case hasPublicRead:
		return TOSACLPublicRead
	default:
		return TOSACLPrivate
	}
}
