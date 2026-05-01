package api

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type S3Bucket struct {
	Name         string
	BucketRegion string
}

type ListBucketsOutput struct {
	Buckets []S3Bucket
}

type GetBucketLocationOutput struct {
	Region string
}

type S3Object struct {
	Key          string
	Size         int64
	LastModified string
	StorageClass string
}

type ListObjectsV2Output struct {
	Objects               []S3Object
	IsTruncated           bool
	NextContinuationToken string
}

type listBucketsResponse struct {
	XMLName xml.Name       `xml:"ListAllMyBucketsResult"`
	Buckets []s3BucketWire `xml:"Buckets>Bucket"`
}

type s3BucketWire struct {
	Name         string `xml:"Name"`
	BucketRegion string `xml:"BucketRegion"`
}

type getBucketLocationResponse struct {
	XMLName xml.Name `xml:"LocationConstraint"`
	Value   string   `xml:",chardata"`
}

type listObjectsV2Response struct {
	XMLName               xml.Name       `xml:"ListBucketResult"`
	IsTruncated           bool           `xml:"IsTruncated"`
	NextContinuationToken string         `xml:"NextContinuationToken"`
	Contents              []s3ObjectWire `xml:"Contents"`
}

type s3ObjectWire struct {
	Key          string `xml:"Key"`
	Size         int64  `xml:"Size"`
	LastModified string `xml:"LastModified"`
	StorageClass string `xml:"StorageClass"`
}

func (c *Client) ListBuckets(ctx context.Context, region string) (ListBucketsOutput, error) {
	var wire listBucketsResponse
	err := c.DoRESTXML(ctx, Request{
		Service:    "s3",
		Region:     region,
		Method:     http.MethodGet,
		Path:       "/",
		Idempotent: true,
	}, &wire)
	if err != nil {
		return ListBucketsOutput{}, err
	}
	out := ListBucketsOutput{
		Buckets: make([]S3Bucket, 0, len(wire.Buckets)),
	}
	for _, bucket := range wire.Buckets {
		name := strings.TrimSpace(bucket.Name)
		if name == "" {
			continue
		}
		out.Buckets = append(out.Buckets, S3Bucket{
			Name:         name,
			BucketRegion: normalizeS3LocationConstraint(bucket.BucketRegion),
		})
	}
	return out, nil
}

func (c *Client) GetBucketLocation(ctx context.Context, region, bucket string) (GetBucketLocationOutput, error) {
	var wire getBucketLocationResponse
	query := url.Values{}
	query.Set("location", "")
	err := c.DoRESTXML(ctx, Request{
		Service:    "s3",
		Region:     region,
		Method:     http.MethodGet,
		Path:       "/" + strings.TrimSpace(bucket),
		Query:      query,
		Idempotent: true,
	}, &wire)
	if err != nil {
		return GetBucketLocationOutput{}, err
	}
	return GetBucketLocationOutput{
		Region: normalizeS3LocationConstraint(wire.Value),
	}, nil
}

func (c *Client) ListObjectsV2(ctx context.Context, region, bucket, continuationToken string, maxKeys int) (ListObjectsV2Output, error) {
	query := url.Values{}
	query.Set("list-type", "2")
	if continuationToken = strings.TrimSpace(continuationToken); continuationToken != "" {
		query.Set("continuation-token", continuationToken)
	}
	if maxKeys > 0 {
		query.Set("max-keys", strconv.Itoa(maxKeys))
	}
	var wire listObjectsV2Response
	err := c.DoRESTXML(ctx, Request{
		Service:    "s3",
		Region:     region,
		Method:     http.MethodGet,
		Path:       "/" + strings.TrimSpace(bucket),
		Query:      query,
		Idempotent: true,
	}, &wire)
	if err != nil {
		return ListObjectsV2Output{}, err
	}
	out := ListObjectsV2Output{
		Objects:               make([]S3Object, 0, len(wire.Contents)),
		IsTruncated:           wire.IsTruncated,
		NextContinuationToken: strings.TrimSpace(wire.NextContinuationToken),
	}
	for _, object := range wire.Contents {
		key := strings.TrimSpace(object.Key)
		if key == "" {
			continue
		}
		out.Objects = append(out.Objects, S3Object{
			Key:          key,
			Size:         object.Size,
			LastModified: strings.TrimSpace(object.LastModified),
			StorageClass: strings.TrimSpace(object.StorageClass),
		})
	}
	return out, nil
}

func normalizeS3LocationConstraint(region string) string {
	switch strings.TrimSpace(region) {
	case "":
		return ""
	case "EU":
		return "eu-west-1"
	default:
		return strings.TrimSpace(region)
	}
}

// GetBucketAclOutput is a slimmed projection of the GetBucketAcl response.
// Only the fields needed to derive a canonical canned-ACL summary are kept.
type GetBucketAclOutput struct {
	Owner  S3Owner
	Grants []S3Grant
}

type S3Owner struct {
	ID          string
	DisplayName string
}

type S3Grant struct {
	GranteeType string
	GranteeID   string
	GranteeURI  string
	Permission  string
}

type getBucketAclResponse struct {
	XMLName           xml.Name `xml:"AccessControlPolicy"`
	Owner             struct {
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

// GetBucketAcl returns the parsed `?acl` response for bucket. Callers collapse
// the grant list into a canned-ACL summary via S3CannedACLFromGrants.
func (c *Client) GetBucketAcl(ctx context.Context, region, bucket string) (GetBucketAclOutput, error) {
	var wire getBucketAclResponse
	query := url.Values{}
	query.Set("acl", "")
	err := c.DoRESTXML(ctx, Request{
		Service:    "s3",
		Region:     region,
		Method:     http.MethodGet,
		Path:       "/" + strings.TrimSpace(bucket),
		Query:      query,
		Idempotent: true,
	}, &wire)
	if err != nil {
		return GetBucketAclOutput{}, err
	}
	out := GetBucketAclOutput{
		Owner: S3Owner{ID: wire.Owner.ID, DisplayName: wire.Owner.DisplayName},
	}
	for _, g := range wire.AccessControlList.Grant {
		out.Grants = append(out.Grants, S3Grant{
			GranteeType: strings.TrimSpace(g.Grantee.Type),
			GranteeID:   strings.TrimSpace(g.Grantee.ID),
			GranteeURI:  strings.TrimSpace(g.Grantee.URI),
			Permission:  strings.ToUpper(strings.TrimSpace(g.Permission)),
		})
	}
	return out, nil
}

// PutBucketAcl sets a canned ACL on bucket via the `x-amz-acl` header.
// Common values: private, public-read, public-read-write, authenticated-read.
func (c *Client) PutBucketAcl(ctx context.Context, region, bucket, cannedACL string) error {
	cannedACL = strings.TrimSpace(cannedACL)
	if cannedACL == "" {
		return fmt.Errorf("aws s3: empty canned acl")
	}
	query := url.Values{}
	query.Set("acl", "")
	headers := http.Header{}
	headers.Set("x-amz-acl", cannedACL)
	return c.DoRESTXML(ctx, Request{
		Service: "s3",
		Region:  region,
		Method:  http.MethodPut,
		Path:    "/" + strings.TrimSpace(bucket),
		Query:   query,
		Headers: headers,
	}, nil)
}

// DeletePublicAccessBlock clears the BlockPublicAcls / IgnorePublicAcls
// settings on bucket so a subsequent canned ACL change actually surfaces.
// New AWS accounts ship with BPA enabled by default; without this call the
// `expose` flow will silently no-op even after PutBucketAcl returns 200.
func (c *Client) DeletePublicAccessBlock(ctx context.Context, region, bucket string) error {
	query := url.Values{}
	query.Set("publicAccessBlock", "")
	return c.DoRESTXML(ctx, Request{
		Service: "s3",
		Region:  region,
		Method:  http.MethodDelete,
		Path:    "/" + strings.TrimSpace(bucket),
		Query:   query,
	}, nil)
}
