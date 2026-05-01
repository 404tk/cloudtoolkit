package cos

import "encoding/xml"

type ListBucketsResponse struct {
	XMLName xml.Name    `xml:"ListAllMyBucketsResult"`
	Buckets []COSBucket `xml:"Buckets>Bucket"`
}

type COSBucket struct {
	Name         string `xml:"Name"`
	Region       string `xml:"Location"`
	CreationDate string `xml:"CreationDate"`
}

type ListObjectsResponse struct {
	XMLName     xml.Name    `xml:"ListBucketResult"`
	Name        string      `xml:"Name"`
	Prefix      string      `xml:"Prefix"`
	Marker      string      `xml:"Marker"`
	NextMarker  string      `xml:"NextMarker"`
	MaxKeys     int         `xml:"MaxKeys"`
	IsTruncated bool        `xml:"IsTruncated"`
	Objects     []COSObject `xml:"Contents"`
}

type COSObject struct {
	Key          string `xml:"Key"`
	Size         int64  `xml:"Size"`
	LastModified string `xml:"LastModified"`
	StorageClass string `xml:"StorageClass"`
}

type errorResponse struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	Resource  string   `xml:"Resource"`
	RequestID string   `xml:"RequestId"`
	TraceID   string   `xml:"TraceId"`
}

// BucketACLResponse maps the body returned by `GET /?acl`. Tencent COS uses
// the same Owner+AccessControlList shape as S3; the canned-ACL view is
// reconstructed from grants because the GET path doesn't echo `x-cos-acl`.
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
