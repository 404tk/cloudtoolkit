package oss

import (
	"encoding/xml"
	"net/url"
)

type ListBucketsResponse struct {
	XMLName     xml.Name    `xml:"ListAllMyBucketsResult"`
	Prefix      string      `xml:"Prefix"`
	Marker      string      `xml:"Marker"`
	MaxKeys     int         `xml:"MaxKeys"`
	IsTruncated bool        `xml:"IsTruncated"`
	NextMarker  string      `xml:"NextMarker"`
	Buckets     []OSSBucket `xml:"Buckets>Bucket"`
}

type OSSBucket struct {
	Name     string `xml:"Name"`
	Location string `xml:"Location"`
	Region   string `xml:"Region"`
}

type ListObjectsResponse struct {
	XMLName               xml.Name    `xml:"ListBucketResult"`
	Name                  string      `xml:"Name"`
	Prefix                string      `xml:"Prefix"`
	StartAfter            string      `xml:"StartAfter"`
	ContinuationToken     string      `xml:"ContinuationToken"`
	MaxKeys               int         `xml:"MaxKeys"`
	Delimiter             string      `xml:"Delimiter"`
	IsTruncated           bool        `xml:"IsTruncated"`
	NextContinuationToken string      `xml:"NextContinuationToken"`
	Objects               []OSSObject `xml:"Contents"`
}

type OSSObject struct {
	Key  string `xml:"Key"`
	Size int64  `xml:"Size"`
}

type errorResponse struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	RequestID string   `xml:"RequestId"`
	HostID    string   `xml:"HostId"`
}

func decodeListObjectsResponse(result *ListObjectsResponse) error {
	if result == nil {
		return nil
	}
	var err error
	result.Prefix, err = url.QueryUnescape(result.Prefix)
	if err != nil {
		return err
	}
	result.StartAfter, err = url.QueryUnescape(result.StartAfter)
	if err != nil {
		return err
	}
	result.Delimiter, err = url.QueryUnescape(result.Delimiter)
	if err != nil {
		return err
	}
	result.NextContinuationToken, err = url.QueryUnescape(result.NextContinuationToken)
	if err != nil {
		return err
	}
	for i := range result.Objects {
		result.Objects[i].Key, err = url.QueryUnescape(result.Objects[i].Key)
		if err != nil {
			return err
		}
	}
	return nil
}
