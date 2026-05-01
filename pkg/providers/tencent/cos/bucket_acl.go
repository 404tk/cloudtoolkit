package cos

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

// Tencent COS canned ACL values, sent via the `x-cos-acl` header.
const (
	COSACLPrivate           = "private"
	COSACLPublicRead        = "public-read"
	COSACLPublicReadWrite   = "public-read-write"
	COSACLAuthenticatedRead = "authenticated-read"
)

// GetBucketACL returns the canned ACL summary derived from the `?acl` grants.
func (c *Client) GetBucketACL(ctx context.Context, bucket, region string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.credential.Validate(); err != nil {
		return "", err
	}
	bucket = strings.TrimSpace(bucket)
	region = strings.TrimSpace(region)
	if bucket == "" || region == "" || region == "all" {
		return "", fmt.Errorf("tencent cos client: empty bucket or region")
	}
	u, err := c.bucketURL(bucket, region)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("acl", "")
	u.RawQuery = q.Encode()

	httpResp, err := c.retryPolicy.Do(ctx, true, func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		if err := Sign(req, c.credential, c.now().UTC()); err != nil {
			return nil, err
		}
		return c.httpClient.Do(req)
	})
	if err != nil {
		return "", err
	}
	if httpResp == nil {
		return "", fmt.Errorf("tencent cos client: empty response")
	}
	defer httpclient.CloseResponse(httpResp)
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return "", fmt.Errorf("read tencent cos response: %w", err)
	}
	if err := decodeError(httpResp, body); err != nil {
		return "", err
	}
	if len(body) == 0 {
		return COSACLPrivate, nil
	}
	var out BucketACLResponse
	if err := xml.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("decode tencent cos response: %w", err)
	}
	return CollapseGrants(out), nil
}

// PutBucketACL sets the canned ACL on bucket via the `x-cos-acl` header.
func (c *Client) PutBucketACL(ctx context.Context, bucket, region, acl string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.credential.Validate(); err != nil {
		return err
	}
	bucket = strings.TrimSpace(bucket)
	region = strings.TrimSpace(region)
	acl = strings.TrimSpace(acl)
	if bucket == "" || region == "" || region == "all" {
		return fmt.Errorf("tencent cos client: empty bucket or region")
	}
	if acl == "" {
		return fmt.Errorf("tencent cos client: empty acl")
	}
	u, err := c.bucketURL(bucket, region)
	if err != nil {
		return err
	}
	q := u.Query()
	q.Set("acl", "")
	u.RawQuery = q.Encode()

	httpResp, err := c.retryPolicy.Do(ctx, false, func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("x-cos-acl", acl)
		if err := Sign(req, c.credential, c.now().UTC()); err != nil {
			return nil, err
		}
		return c.httpClient.Do(req)
	})
	if err != nil {
		return err
	}
	if httpResp == nil {
		return fmt.Errorf("tencent cos client: empty response")
	}
	defer httpclient.CloseResponse(httpResp)
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("read tencent cos response: %w", err)
	}
	return decodeError(httpResp, body)
}

// AuditBucketACL enumerates buckets and returns their canned ACL summary.
func (d *Driver) AuditBucketACL(ctx context.Context, bucket string) ([]schema.BucketACLEntry, error) {
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
		acl, err := d.client().GetBucketACL(ctx, s.BucketName, s.Region)
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
	acl := NormalizeCOSACL(level)
	if acl == "" || acl == COSACLPrivate {
		acl = COSACLPublicRead
	}
	region, err := d.bucketRegion(ctx, bucket)
	if err != nil {
		return "", err
	}
	if err := d.client().PutBucketACL(ctx, bucket, region, acl); err != nil {
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
	return d.client().PutBucketACL(ctx, bucket, region, COSACLPrivate)
}

func (d *Driver) bucketRegion(ctx context.Context, bucket string) (string, error) {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return "", fmt.Errorf("tencent cos: empty bucket")
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
	return "", fmt.Errorf("tencent cos: region for bucket %q not found", bucket)
}

// NormalizeCOSACL maps user-friendly aliases to canned COS ACL values.
func NormalizeCOSACL(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", "private":
		return COSACLPrivate
	case "public-read", "publicread", "blob", "container":
		return COSACLPublicRead
	case "public-read-write", "publicreadwrite", "rw", "writable":
		return COSACLPublicReadWrite
	case "authenticated-read", "authenticatedread":
		return COSACLAuthenticatedRead
	}
	return strings.ToLower(strings.TrimSpace(level))
}

// CollapseGrants reduces the parsed ACL grant list back into a canned label.
func CollapseGrants(out BucketACLResponse) string {
	publicRead := false
	publicWrite := false
	for _, g := range out.AccessControlList.Grant {
		uri := strings.ToLower(strings.TrimSpace(g.Grantee.URI))
		if !strings.Contains(uri, "qcs::cam::anyone:anyone") &&
			!strings.Contains(uri, "groups/global/allusers") {
			continue
		}
		switch strings.ToUpper(strings.TrimSpace(g.Permission)) {
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
		return COSACLPublicReadWrite
	case publicRead:
		return COSACLPublicRead
	default:
		return COSACLPrivate
	}
}
