package oss

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	awsapi "github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// JDCloud OSS canned ACL values, sent via the `x-amz-acl` header (S3-style).
const (
	OSSACLPrivate           = "private"
	OSSACLPublicRead        = "public-read"
	OSSACLPublicReadWrite   = "public-read-write"
	OSSACLAuthenticatedRead = "authenticated-read"
)

// BucketACLOutput is the parsed `?acl` response. JDCloud OSS exposes the
// S3-compatible Owner+AccessControlList shape.
type BucketACLOutput struct {
	Owner  Owner
	Grants []Grant
}

type Owner struct {
	ID          string
	DisplayName string
}

type Grant struct {
	GranteeType string
	GranteeID   string
	GranteeURI  string
	Permission  string
}

type bucketACLResponse struct {
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

// GetBucketAcl fetches the parsed ACL response for bucket.
func (c *Client) GetBucketAcl(ctx context.Context, bucket, region string) (BucketACLOutput, error) {
	if c == nil || c.api == nil {
		return BucketACLOutput{}, errors.New("jdcloud oss: nil object client")
	}
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return BucketACLOutput{}, fmt.Errorf("jdcloud oss: empty bucket")
	}
	query := url.Values{}
	query.Set("acl", "")
	var wire bucketACLResponse
	err := c.api.DoRESTXML(ctx, awsapi.Request{
		Service:    "s3",
		Region:     normalizeBucketRegion(region),
		Method:     http.MethodGet,
		Path:       bucketPath(bucket),
		Query:      query,
		Host:       serviceHost(region),
		Idempotent: true,
	}, &wire)
	if err != nil {
		return BucketACLOutput{}, err
	}
	out := BucketACLOutput{
		Owner: Owner{ID: wire.Owner.ID, DisplayName: wire.Owner.DisplayName},
	}
	for _, g := range wire.AccessControlList.Grant {
		out.Grants = append(out.Grants, Grant{
			GranteeType: strings.TrimSpace(g.Grantee.Type),
			GranteeID:   strings.TrimSpace(g.Grantee.ID),
			GranteeURI:  strings.TrimSpace(g.Grantee.URI),
			Permission:  strings.ToUpper(strings.TrimSpace(g.Permission)),
		})
	}
	return out, nil
}

// PutBucketAcl sets the canned ACL on bucket via the `x-amz-acl` header.
func (c *Client) PutBucketAcl(ctx context.Context, bucket, region, cannedACL string) error {
	if c == nil || c.api == nil {
		return errors.New("jdcloud oss: nil object client")
	}
	bucket = strings.TrimSpace(bucket)
	cannedACL = strings.TrimSpace(cannedACL)
	if bucket == "" {
		return fmt.Errorf("jdcloud oss: empty bucket")
	}
	if cannedACL == "" {
		return fmt.Errorf("jdcloud oss: empty canned acl")
	}
	query := url.Values{}
	query.Set("acl", "")
	headers := http.Header{}
	headers.Set("x-amz-acl", cannedACL)
	return c.api.DoRESTXML(ctx, awsapi.Request{
		Service: "s3",
		Region:  normalizeBucketRegion(region),
		Method:  http.MethodPut,
		Path:    bucketPath(bucket),
		Query:   query,
		Headers: headers,
		Host:    serviceHost(region),
	}, nil)
}

// AuditBucketACL enumerates buckets and returns canned ACL state for each.
func (d *Driver) AuditBucketACL(ctx context.Context, bucket string) ([]schema.BucketACLEntry, error) {
	storages, err := d.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}
	bucket = strings.TrimSpace(bucket)
	out := make([]schema.BucketACLEntry, 0, len(storages))
	client, err := d.objectClient()
	if err != nil {
		return out, err
	}
	for _, s := range storages {
		if bucket != "" && s.BucketName != bucket {
			continue
		}
		region := s.Region
		if region == "" {
			region = d.normalizedRegion()
		}
		acl, err := client.GetBucketAcl(ctx, s.BucketName, region)
		if err != nil {
			return out, fmt.Errorf("get acl for %s: %w", s.BucketName, err)
		}
		out = append(out, schema.BucketACLEntry{
			Container: s.BucketName,
			Level:     CannedACLFromGrants(acl),
		})
	}
	return out, nil
}

// ExposeBucket sets bucket public-readable (defaults to public-read).
func (d *Driver) ExposeBucket(ctx context.Context, bucket, level string) (string, error) {
	cannedACL := NormalizeOSSACL(level)
	if cannedACL == "" || cannedACL == OSSACLPrivate {
		cannedACL = OSSACLPublicRead
	}
	region, err := d.bucketRegion(ctx, bucket)
	if err != nil {
		return "", err
	}
	client, err := d.objectClient()
	if err != nil {
		return "", err
	}
	if err := client.PutBucketAcl(ctx, bucket, region, cannedACL); err != nil {
		return "", err
	}
	return cannedACL, nil
}

// UnexposeBucket reverts bucket to private.
func (d *Driver) UnexposeBucket(ctx context.Context, bucket string) error {
	region, err := d.bucketRegion(ctx, bucket)
	if err != nil {
		return err
	}
	client, err := d.objectClient()
	if err != nil {
		return err
	}
	return client.PutBucketAcl(ctx, bucket, region, OSSACLPrivate)
}

func (d *Driver) bucketRegion(ctx context.Context, bucket string) (string, error) {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return "", fmt.Errorf("jdcloud oss: empty bucket")
	}
	region, err := d.ResolveBucketRegion(ctx, bucket)
	if err == nil && region != "" {
		return region, nil
	}
	if r := d.normalizedRegion(); r != "" && r != "all" {
		return r, nil
	}
	return defaultBucketRegion, nil
}

// NormalizeOSSACL maps user-friendly aliases to canned JDCloud OSS ACL values.
func NormalizeOSSACL(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", "private":
		return OSSACLPrivate
	case "public-read", "publicread", "blob", "container":
		return OSSACLPublicRead
	case "public-read-write", "publicreadwrite", "rw", "writable":
		return OSSACLPublicReadWrite
	case "authenticated-read", "authenticatedread":
		return OSSACLAuthenticatedRead
	}
	return strings.ToLower(strings.TrimSpace(level))
}

// CannedACLFromGrants reduces a parsed grant list to a canned ACL label.
func CannedACLFromGrants(out BucketACLOutput) string {
	publicRead := false
	publicWrite := false
	for _, g := range out.Grants {
		uri := strings.ToLower(strings.TrimSpace(g.GranteeURI))
		if !strings.Contains(uri, "groups/global/allusers") &&
			!strings.Contains(uri, "everyone") {
			continue
		}
		switch g.Permission {
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
		return OSSACLPublicReadWrite
	case publicRead:
		return OSSACLPublicRead
	default:
		return OSSACLPrivate
	}
}
