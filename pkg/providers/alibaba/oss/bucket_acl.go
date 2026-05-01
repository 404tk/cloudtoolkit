package oss

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/internal/httpclient"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// Alibaba OSS canned ACL values. Levels accepted by the bucket-acl-check
// payload map onto these via NormalizeOSSACL.
const (
	OSSACLPrivate          = "private"
	OSSACLPublicRead       = "public-read"
	OSSACLPublicReadWrite  = "public-read-write"
)

// GetBucketACL returns the canned ACL grant ("private" / "public-read" /
// "public-read-write") currently set on bucket.
func (c *Client) GetBucketACL(ctx context.Context, bucket, region string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.credential.Validate(); err != nil {
		return "", err
	}
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return "", fmt.Errorf("alibaba oss client: empty bucket")
	}
	region = strings.TrimSpace(region)
	if region == "" || region == "all" {
		return "", fmt.Errorf("alibaba oss client: empty region")
	}

	u, err := c.bucketURL(bucket, region)
	if err != nil {
		return "", err
	}
	query := u.Query()
	query.Set("acl", "")
	u.RawQuery = query.Encode()

	httpResp, err := c.retryPolicy.Do(ctx, true, func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		if err := Sign(req, c.credential, bucket, c.now().UTC()); err != nil {
			return nil, err
		}
		return c.httpClient.Do(req)
	})
	if err != nil {
		return "", err
	}
	if httpResp == nil {
		return "", fmt.Errorf("alibaba oss client: empty response")
	}
	defer httpclient.CloseResponse(httpResp)

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return "", fmt.Errorf("read alibaba oss response: %w", err)
	}
	if err := decodeError(httpResp, body); err != nil {
		return "", err
	}
	var out BucketACLResponse
	if len(body) == 0 {
		return "", nil
	}
	if err := xml.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("decode alibaba oss response: %w", err)
	}
	return strings.TrimSpace(out.AccessControlList.Grant), nil
}

// PutBucketACL sets the canned ACL on bucket. acl must be one of the OSSACL*
// constants; the value is sent via the `x-oss-acl` header per OSS spec.
func (c *Client) PutBucketACL(ctx context.Context, bucket, region, acl string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.credential.Validate(); err != nil {
		return err
	}
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return fmt.Errorf("alibaba oss client: empty bucket")
	}
	region = strings.TrimSpace(region)
	if region == "" || region == "all" {
		return fmt.Errorf("alibaba oss client: empty region")
	}
	acl = strings.TrimSpace(acl)
	if acl == "" {
		return fmt.Errorf("alibaba oss client: empty acl")
	}

	u, err := c.bucketURL(bucket, region)
	if err != nil {
		return err
	}
	query := u.Query()
	query.Set("acl", "")
	u.RawQuery = query.Encode()

	httpResp, err := c.retryPolicy.Do(ctx, false, func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("x-oss-acl", acl)
		if err := Sign(req, c.credential, bucket, c.now().UTC()); err != nil {
			return nil, err
		}
		return c.httpClient.Do(req)
	})
	if err != nil {
		return err
	}
	if httpResp == nil {
		return fmt.Errorf("alibaba oss client: empty response")
	}
	defer httpclient.CloseResponse(httpResp)

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("read alibaba oss response: %w", err)
	}
	return decodeError(httpResp, body)
}

// AuditBucketACL enumerates buckets in scope and returns their canned ACL
// state. When bucket is empty all buckets are audited; otherwise the named
// bucket is audited if found.
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
		grant, err := client.GetBucketACL(ctx, s.BucketName, region)
		if err != nil {
			return out, fmt.Errorf("get acl for %s: %w", s.BucketName, err)
		}
		if grant == "" {
			grant = OSSACLPrivate
		}
		out = append(out, schema.BucketACLEntry{
			Container: s.BucketName,
			Level:     grant,
		})
	}
	return out, nil
}

// ExposeBucket sets bucket public-readable. level overrides the default
// `public-read` (e.g. `public-read-write`) if the caller wants a stronger
// expose.
func (d *Driver) ExposeBucket(ctx context.Context, bucket, level string) (string, error) {
	acl := NormalizeOSSACL(level)
	if acl == "" || acl == OSSACLPrivate {
		acl = OSSACLPublicRead
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

// UnexposeBucket reverts bucket to `private`.
func (d *Driver) UnexposeBucket(ctx context.Context, bucket string) error {
	region, err := d.bucketRegion(ctx, bucket)
	if err != nil {
		return err
	}
	client, err := d.NewClient()
	if err != nil {
		return err
	}
	return client.PutBucketACL(ctx, bucket, region, OSSACLPrivate)
}

func (d *Driver) bucketRegion(ctx context.Context, bucket string) (string, error) {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return "", fmt.Errorf("alibaba oss: empty bucket")
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
	return "", fmt.Errorf("alibaba oss: region for bucket %q not found", bucket)
}

// NormalizeOSSACL maps user-friendly aliases to the canned OSS ACL values.
func NormalizeOSSACL(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", "private":
		return OSSACLPrivate
	case "public-read", "publicread", "blob", "container":
		return OSSACLPublicRead
	case "public-read-write", "publicreadwrite", "rw", "writable":
		return OSSACLPublicReadWrite
	}
	return strings.ToLower(strings.TrimSpace(level))
}
