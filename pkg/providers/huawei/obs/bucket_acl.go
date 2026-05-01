package obs

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/endpoint"
	"github.com/404tk/cloudtoolkit/pkg/providers/internal/httpclient"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// Huawei OBS canned ACL values, sent via the `x-obs-acl` header.
const (
	OBSACLPrivate                 = "private"
	OBSACLPublicRead              = "public-read"
	OBSACLPublicReadWrite         = "public-read-write"
	OBSACLPublicReadDelivered     = "public-read-delivered"
	OBSACLPublicReadWriteDelivered = "public-read-write-delivered"
)

// BucketACLResponse maps the body returned by `GET /?acl`. OBS exposes the
// same Owner+AccessControlList shape as S3.
type BucketACLResponse struct {
	XMLName xml.Name `xml:"AccessControlPolicy"`
	Owner   struct {
		ID          string `xml:"ID"`
		DisplayName string `xml:"DisplayName"`
	} `xml:"Owner"`
	AccessControlList struct {
		Grant []struct {
			Grantee struct {
				Type string `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr"`
				ID   string `xml:"ID"`
				URI  string `xml:"URI"`
			} `xml:"Grantee"`
			Permission string `xml:"Permission"`
		} `xml:"Grant"`
	} `xml:"AccessControlList"`
}

// GetBucketACL returns the canned ACL summary for bucket. Grants are folded
// back into a canonical canned-ACL string for the audit table.
func (c *Client) GetBucketACL(ctx context.Context, bucket, region string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.credential.Validate(); err != nil {
		return "", err
	}
	bucket = strings.TrimSpace(bucket)
	region = strings.TrimSpace(region)
	if bucket == "" || region == "" {
		return "", fmt.Errorf("huawei obs client: empty bucket or region")
	}

	rawURL := endpoint.For("obs", region, c.credential.Intl)
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("huawei obs client: invalid endpoint %q: %w", rawURL, err)
	}
	u.Path = "/" + bucket
	query := url.Values{}
	query.Set("acl", "")
	u.RawQuery = query.Encode()

	signed, err := Sign(&SignRequest{
		Method:    http.MethodGet,
		Path:      "/" + bucket,
		Query:     query,
		Scheme:    authSchemeV2,
		AccessKey: c.credential.AK,
		SecretKey: c.credential.SK,
		Timestamp: c.now().UTC(),
	})
	if err != nil {
		return "", err
	}

	httpResp, err := c.retryPolicy.Do(ctx, true, func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Host = u.Host
		req.Header = signed.Clone()
		return c.httpClient.Do(req)
	})
	if err != nil {
		return "", err
	}
	if httpResp == nil {
		return "", fmt.Errorf("huawei obs client: empty response")
	}
	defer httpclient.CloseResponse(httpResp)

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return "", fmt.Errorf("read huawei obs response: %w", err)
	}
	if err := decodeError(httpResp.StatusCode, httpResp.Header, body); err != nil {
		return "", err
	}
	if len(body) == 0 {
		return OBSACLPrivate, nil
	}
	var out BucketACLResponse
	if err := xml.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("decode huawei obs response: %w", err)
	}
	return CollapseGrants(out), nil
}

// PutBucketACL sets a canned ACL on bucket via the `x-obs-acl` header.
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
	if bucket == "" || region == "" {
		return fmt.Errorf("huawei obs client: empty bucket or region")
	}
	if acl == "" {
		return fmt.Errorf("huawei obs client: empty acl")
	}

	rawURL := endpoint.For("obs", region, c.credential.Intl)
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("huawei obs client: invalid endpoint %q: %w", rawURL, err)
	}
	u.Path = "/" + bucket
	query := url.Values{}
	query.Set("acl", "")
	u.RawQuery = query.Encode()

	headers := http.Header{}
	headers.Set("x-obs-acl", acl)

	signed, err := Sign(&SignRequest{
		Method:    http.MethodPut,
		Path:      "/" + bucket,
		Query:     query,
		Headers:   headers,
		Scheme:    authSchemeV2,
		AccessKey: c.credential.AK,
		SecretKey: c.credential.SK,
		Timestamp: c.now().UTC(),
	})
	if err != nil {
		return err
	}

	httpResp, err := c.retryPolicy.Do(ctx, false, func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Host = u.Host
		req.Header = signed.Clone()
		return c.httpClient.Do(req)
	})
	if err != nil {
		return err
	}
	if httpResp == nil {
		return fmt.Errorf("huawei obs client: empty response")
	}
	defer httpclient.CloseResponse(httpResp)

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("read huawei obs response: %w", err)
	}
	return decodeError(httpResp.StatusCode, httpResp.Header, body)
}

// AuditBucketACL enumerates buckets in scope and returns the canned ACL state.
func (d *Driver) AuditBucketACL(ctx context.Context, bucket string) ([]schema.BucketACLEntry, error) {
	storages, err := d.GetBuckets(ctx)
	if err != nil {
		return nil, err
	}
	bucket = strings.TrimSpace(bucket)
	out := make([]schema.BucketACLEntry, 0, len(storages))
	client := d.client()
	for _, s := range storages {
		if bucket != "" && s.BucketName != bucket {
			continue
		}
		region := s.Region
		if region == "" {
			region = d.requestRegion()
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

// ExposeBucket sets bucket public-readable (defaults to public-read).
func (d *Driver) ExposeBucket(ctx context.Context, bucket, level string) (string, error) {
	acl := NormalizeOBSACL(level)
	if acl == "" || acl == OBSACLPrivate {
		acl = OBSACLPublicRead
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

// UnexposeBucket reverts bucket to private.
func (d *Driver) UnexposeBucket(ctx context.Context, bucket string) error {
	region, err := d.bucketRegion(ctx, bucket)
	if err != nil {
		return err
	}
	return d.client().PutBucketACL(ctx, bucket, region, OBSACLPrivate)
}

func (d *Driver) bucketRegion(ctx context.Context, bucket string) (string, error) {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return "", fmt.Errorf("huawei obs: empty bucket")
	}
	storages, err := d.GetBuckets(ctx)
	if err != nil {
		return "", err
	}
	for _, s := range storages {
		if s.BucketName == bucket && s.Region != "" {
			return s.Region, nil
		}
	}
	return d.requestRegion(), nil
}

// NormalizeOBSACL maps user-friendly aliases to canned OBS ACL values.
func NormalizeOBSACL(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", "private":
		return OBSACLPrivate
	case "public-read", "publicread", "blob", "container":
		return OBSACLPublicRead
	case "public-read-write", "publicreadwrite", "rw", "writable":
		return OBSACLPublicReadWrite
	case "public-read-delivered":
		return OBSACLPublicReadDelivered
	case "public-read-write-delivered":
		return OBSACLPublicReadWriteDelivered
	}
	return strings.ToLower(strings.TrimSpace(level))
}

// CollapseGrants reduces a parsed grant list into a canned ACL label.
func CollapseGrants(out BucketACLResponse) string {
	publicRead := false
	publicWrite := false
	for _, g := range out.AccessControlList.Grant {
		uri := strings.ToLower(strings.TrimSpace(g.Grantee.URI))
		if !strings.Contains(uri, "everyone") &&
			!strings.Contains(uri, "groups/global/allusers") &&
			g.Grantee.Type != "Group" {
			continue
		}
		if g.Grantee.Type == "Group" && g.Grantee.URI == "" {
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
		return OBSACLPublicReadWrite
	case publicRead:
		return OBSACLPublicRead
	default:
		return OBSACLPrivate
	}
}
