package api

import (
	"context"
	"encoding/xml"
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
	Key  string
	Size int64
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
	Key  string `xml:"Key"`
	Size int64  `xml:"Size"`
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
			Key:  key,
			Size: object.Size,
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
