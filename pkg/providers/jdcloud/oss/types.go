package oss

import (
	"encoding/xml"
	"fmt"
	"strings"
)

type ListObjectsV2Output struct {
	Objects               []Object
	IsTruncated           bool
	NextContinuationToken string
}

type Object struct {
	Key          string
	Size         int64
	LastModified string
	StorageClass string
}

type listObjectsV2Response struct {
	XMLName               xml.Name     `xml:"ListBucketResult"`
	IsTruncated           bool         `xml:"IsTruncated"`
	NextContinuationToken string       `xml:"NextContinuationToken"`
	Contents              []objectWire `xml:"Contents"`
}

type objectWire struct {
	Key          string `xml:"Key"`
	Size         int64  `xml:"Size"`
	LastModified string `xml:"LastModified"`
	StorageClass string `xml:"StorageClass"`
}

func normalizeBucketRegion(region string) string {
	region = strings.TrimSpace(region)
	if region == "" || strings.EqualFold(region, "all") {
		return defaultBucketRegion
	}
	return region
}

func serviceHost(region string) string {
	return fmt.Sprintf("s3.%s.jdcloud-oss.com", normalizeBucketRegion(region))
}

func bucketPath(bucket string) string {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return "/"
	}
	return "/" + bucket
}
