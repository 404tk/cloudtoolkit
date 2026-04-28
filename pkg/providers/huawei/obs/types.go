package obs

import "encoding/xml"

type ListBucketsResponse struct {
	XMLName xml.Name    `xml:"ListAllMyBucketsResult"`
	Buckets []OBSBucket `xml:"Buckets>Bucket"`
}

type ListObjectsResponse struct {
	XMLName     xml.Name    `xml:"ListBucketResult"`
	IsTruncated bool        `xml:"IsTruncated"`
	Marker      string      `xml:"Marker"`
	NextMarker  string      `xml:"NextMarker"`
	MaxKeys     int         `xml:"MaxKeys"`
	Name        string      `xml:"Name"`
	Prefix      string      `xml:"Prefix"`
	Objects     []OBSObject `xml:"Contents"`
}

type OBSBucket struct {
	Name     string `xml:"Name"`
	Location string `xml:"Location"`
}

type OBSObject struct {
	Key          string `xml:"Key"`
	Size         int64  `xml:"Size"`
	LastModified string `xml:"LastModified"`
	StorageClass string `xml:"StorageClass"`
}

type errorResponse struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	RequestID string   `xml:"RequestId"`
	HostID    string   `xml:"HostId"`
}
