package tos

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	volcapi "github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
)

type ListBucketsOutput struct {
	Buckets []Bucket `json:"Buckets"`
	Owner   struct {
		ID string `json:"ID"`
	} `json:"Owner"`
}

type Bucket struct {
	Name             string `json:"Name"`
	CreationDate     string `json:"CreationDate"`
	Location         string `json:"Location"`
	ExtranetEndpoint string `json:"ExtranetEndpoint"`
	IntranetEndpoint string `json:"IntranetEndpoint"`
	ProjectName      string `json:"ProjectName"`
	BucketType       string `json:"BucketType"`
}

type ListObjectsV2Output struct {
	Name                  string       `json:"Name"`
	Prefix                string       `json:"Prefix"`
	MaxKeys               int          `json:"MaxKeys"`
	Delimiter             string       `json:"Delimiter"`
	EncodingType          string       `json:"EncodingType"`
	IsTruncated           bool         `json:"IsTruncated"`
	ContinuationToken     string       `json:"ContinuationToken"`
	NextContinuationToken string       `json:"NextContinuationToken"`
	Contents              []BucketItem `json:"Contents"`
}

type BucketItem struct {
	Key           string `json:"Key"`
	LastModified  string `json:"LastModified"`
	ETag          string `json:"ETag"`
	Size          int64  `json:"Size"`
	StorageClass  string `json:"StorageClass"`
	Type          string `json:"Type"`
	HashCrc64ECMA string `json:"HashCrc64ecma"`
}

func decodeError(statusCode int, headers http.Header, body []byte) error {
	if statusCode < http.StatusBadRequest {
		return nil
	}

	var envelope struct {
		Code      string `json:"Code"`
		Message   string `json:"Message"`
		RequestID string `json:"RequestId"`
	}
	_ = json.Unmarshal(body, &envelope)

	requestID := strings.TrimSpace(envelope.RequestID)
	if requestID == "" {
		requestID = strings.TrimSpace(headers.Get("X-Tos-Request-Id"))
	}

	message := strings.TrimSpace(envelope.Message)
	if message == "" {
		message = strings.TrimSpace(string(body))
	}
	if message == "" {
		message = http.StatusText(statusCode)
	}

	return &volcapi.APIError{
		HTTPStatus: statusCode,
		Code:       strings.TrimSpace(envelope.Code),
		Message:    fmt.Sprintf("tos: %s", message),
		RequestID:  requestID,
	}
}
