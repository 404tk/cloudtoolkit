package replay

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	awsapi "github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	awsauth "github.com/404tk/cloudtoolkit/pkg/providers/aws/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

func (t *transport) handleOSS(req *http.Request) (*http.Response, error) {
	method := strings.ToUpper(req.Method)
	path := req.URL.Path
	switch {
	case method == http.MethodGet && path == "/v1/regions/cn-north-1/buckets":
		return t.handleListBuckets(req)
	case method == http.MethodHead && strings.HasPrefix(path, "/v1/regions/") && strings.Contains(path, "/buckets/"):
		return t.handleHeadBucket(req)
	}
	return apiErrorResponse(req, http.StatusNotFound, "NotFound",
		fmt.Sprintf("unsupported oss path: %s %s", method, path)), nil
}

func (t *transport) handleOSSDataPlane(req *http.Request, host string, body []byte) (*http.Response, error) {
	switch verifyS3Auth(req, host, body) {
	case demoreplay.AuthInvalidAccessKey:
		return ossErrorResponse(req, http.StatusForbidden, "InvalidAccessKeyId",
			"The Access Key Id you provided does not exist in our records."), nil
	case demoreplay.AuthInvalidSignature:
		return ossErrorResponse(req, http.StatusForbidden, "SignatureDoesNotMatch",
			"The request signature we calculated does not match the signature you provided."), nil
	}

	bucketName := strings.TrimSpace(strings.TrimPrefix(req.URL.Path, "/"))
	if bucketName == "" {
		return ossErrorResponse(req, http.StatusBadRequest, "InvalidBucketName", "bucket name required"), nil
	}
	region := ossRegionFromHost(host)
	query := req.URL.Query()
	if query.Has("acl") {
		return t.handleOSSBucketACL(req, bucketName, region)
	}
	if req.Method != http.MethodGet {
		return ossErrorResponse(req, http.StatusMethodNotAllowed, "MethodNotAllowed", "unsupported object method"), nil
	}
	if query.Get("list-type") != "2" {
		return ossErrorResponse(req, http.StatusBadRequest, "InvalidArgument", "only list-type=2 is supported in replay"), nil
	}
	return t.handleOSSListObjectsV2(req, bucketName, region, query)
}

func (t *transport) handleListBuckets(req *http.Request) (*http.Response, error) {
	resp := api.ListBucketsResponse{RequestID: "req-replay-oss-list"}
	for _, bucket := range demoBuckets {
		resp.Result.Buckets = append(resp.Result.Buckets, api.Bucket{Name: bucket.Name})
	}
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

func (t *transport) handleHeadBucket(req *http.Request) (*http.Response, error) {
	rest := strings.TrimPrefix(req.URL.Path, "/v1/regions/")
	parts := strings.SplitN(rest, "/buckets/", 2)
	if len(parts) != 2 {
		return apiErrorResponse(req, http.StatusBadRequest, "InvalidPath",
			"malformed bucket head path"), nil
	}
	region := strings.TrimSpace(parts[0])
	bucketName := strings.TrimSpace(parts[1])
	bucket, ok := findBucket(bucketName)
	if ok {
		if region == bucket.Region {
			resp := demoreplay.JSONResponse(req, http.StatusOK, struct {
				RequestID string `json:"requestId"`
			}{RequestID: "req-replay-oss-head"})
			return resp, nil
		}
		return apiErrorResponse(req, http.StatusNotFound, "NoSuchBucket",
			"bucket exists in a different region"), nil
	}
	return apiErrorResponse(req, http.StatusNotFound, "NoSuchBucket",
		fmt.Sprintf("bucket %s not found", bucketName)), nil
}

func (t *transport) handleOSSBucketACL(req *http.Request, bucketName, region string) (*http.Response, error) {
	bucket, ok := findBucket(bucketName)
	if !ok {
		return ossErrorResponse(req, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist."), nil
	}
	if region != "" && region != bucket.Region {
		return ossErrorResponse(req, http.StatusMovedPermanently, "PermanentRedirect",
			"The bucket you are attempting to access must be addressed using the specified endpoint."), nil
	}
	switch req.Method {
	case http.MethodGet:
		t.mu.Lock()
		acl, ok := t.bucketACL[bucketName]
		t.mu.Unlock()
		if !ok {
			acl = "private"
		}
		resp := bucketACLResponse{}
		resp.Owner.ID = "ctk-demo-owner"
		resp.Owner.DisplayName = "ctk-demo"
		switch acl {
		case "public-read":
			resp.AccessControlList.Grant = append(resp.AccessControlList.Grant, bucketACLGrant{
				Grantee:    bucketACLGrantee{Type: "Group", URI: "http://acs.amazonaws.com/groups/global/AllUsers"},
				Permission: "READ",
			})
		case "public-read-write":
			resp.AccessControlList.Grant = append(resp.AccessControlList.Grant, bucketACLGrant{
				Grantee:    bucketACLGrantee{Type: "Group", URI: "http://acs.amazonaws.com/groups/global/AllUsers"},
				Permission: "FULL_CONTROL",
			})
		}
		return ossXMLResponse(req, http.StatusOK, resp), nil
	case http.MethodPut:
		acl := strings.TrimSpace(req.Header.Get("x-amz-acl"))
		if acl == "" {
			return ossErrorResponse(req, http.StatusBadRequest, "InvalidArgument", "missing x-amz-acl header"), nil
		}
		t.mu.Lock()
		t.bucketACL[bucketName] = acl
		t.mu.Unlock()
		return ossXMLResponse(req, http.StatusOK, struct{}{}), nil
	}
	return ossErrorResponse(req, http.StatusMethodNotAllowed, "MethodNotAllowed", "unsupported acl method"), nil
}

func (t *transport) handleOSSListObjectsV2(req *http.Request, bucketName, region string, query url.Values) (*http.Response, error) {
	bucket, ok := findBucket(bucketName)
	if !ok {
		return ossErrorResponse(req, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist."), nil
	}
	if region != "" && region != bucket.Region {
		return ossErrorResponse(req, http.StatusMovedPermanently, "PermanentRedirect",
			"The bucket you are attempting to access must be addressed using the specified endpoint."), nil
	}

	maxKeys := demoreplay.ParseInt(query.Get("max-keys"), 1000)
	token := strings.TrimSpace(query.Get("continuation-token"))
	start := 0
	if token != "" {
		for i, object := range bucket.Objects {
			if object.Key > token {
				start = i
				break
			}
			start = i + 1
		}
	}
	end := start + maxKeys
	if end > len(bucket.Objects) {
		end = len(bucket.Objects)
	}
	page := bucket.Objects[start:end]

	resp := listBucketResult{
		Name:        bucket.Name,
		IsTruncated: end < len(bucket.Objects),
	}
	if resp.IsTruncated && len(page) > 0 {
		resp.NextContinuationToken = page[len(page)-1].Key
	}
	for _, object := range page {
		resp.Contents = append(resp.Contents, listBucketObject{
			Key:          object.Key,
			Size:         object.Size,
			LastModified: object.LastModified,
			StorageClass: object.StorageClass,
		})
	}
	return ossXMLResponse(req, http.StatusOK, resp), nil
}

type listBucketResult struct {
	XMLName               xml.Name           `xml:"ListBucketResult"`
	Name                  string             `xml:"Name,omitempty"`
	IsTruncated           bool               `xml:"IsTruncated"`
	NextContinuationToken string             `xml:"NextContinuationToken,omitempty"`
	Contents              []listBucketObject `xml:"Contents"`
}

type listBucketObject struct {
	Key          string `xml:"Key"`
	Size         int64  `xml:"Size"`
	LastModified string `xml:"LastModified,omitempty"`
	StorageClass string `xml:"StorageClass,omitempty"`
}

type bucketACLResponse struct {
	XMLName xml.Name `xml:"AccessControlPolicy"`
	Owner   struct {
		ID          string `xml:"ID"`
		DisplayName string `xml:"DisplayName"`
	} `xml:"Owner"`
	AccessControlList struct {
		Grant []bucketACLGrant `xml:"Grant"`
	} `xml:"AccessControlList"`
}

type bucketACLGrant struct {
	Grantee    bucketACLGrantee `xml:"Grantee"`
	Permission string           `xml:"Permission"`
}

type bucketACLGrantee struct {
	Type string `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr"`
	ID   string `xml:"ID,omitempty"`
	URI  string `xml:"URI,omitempty"`
}

type ossErrorPayload struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	RequestID string   `xml:"RequestId"`
	HostID    string   `xml:"HostId"`
}

func ossXMLResponse(req *http.Request, statusCode int, payload any) *http.Response {
	body, _ := xml.Marshal(payload)
	resp := demoreplay.Response(req, statusCode, "application/xml", body)
	resp.Header.Set("X-Amz-Request-Id", fmt.Sprintf("req-replay-jdcloud-oss-%d", statusCode))
	return resp
}

func ossErrorResponse(req *http.Request, statusCode int, code, message string) *http.Response {
	payload := ossErrorPayload{
		Code:      strings.TrimSpace(code),
		Message:   strings.TrimSpace(message),
		RequestID: fmt.Sprintf("req-replay-jdcloud-oss-%d", statusCode),
		HostID:    "replay",
	}
	body, _ := xml.Marshal(payload)
	resp := demoreplay.Response(req, statusCode, "application/xml", body)
	resp.Header.Set("X-Amz-Request-Id", payload.RequestID)
	return resp
}

func verifyS3Auth(req *http.Request, host string, body []byte) demoreplay.AuthFailureKind {
	authHeader := strings.TrimSpace(req.Header.Get("Authorization"))
	parsed, ok := parseSigV4Auth(authHeader)
	if !ok {
		return demoreplay.AuthInvalidSignature
	}
	if parsed.AccessKey != demoCredentials.AccessKey {
		return demoreplay.AuthInvalidAccessKey
	}
	amzDate := strings.TrimSpace(req.Header.Get("X-Amz-Date"))
	timestamp, err := time.Parse("20060102T150405Z", amzDate)
	if err != nil {
		return demoreplay.AuthInvalidSignature
	}
	extra := req.Header.Clone()
	extra.Del("Authorization")
	extra.Del("Host")
	signed, err := (awsapi.SigV4Signer{}).Sign(awsauth.New(demoCredentials.AccessKey, demoCredentials.SecretKey, ""), awsapi.SignInput{
		Method:      req.Method,
		Service:     parsed.Service,
		Region:      parsed.Region,
		Host:        host,
		Path:        req.URL.Path,
		Query:       req.URL.Query(),
		ContentType: strings.TrimSpace(req.Header.Get("Content-Type")),
		Payload:     body,
		Timestamp:   timestamp,
		Headers:     extra,
	})
	if err != nil {
		return demoreplay.AuthInvalidSignature
	}
	if !demoreplay.SubtleEqual(strings.TrimSpace(signed.Authorization), authHeader) {
		return demoreplay.AuthInvalidSignature
	}
	return demoreplay.AuthOK
}

type sigV4AuthHeader struct {
	AccessKey string
	Region    string
	Service   string
}

func parseSigV4Auth(value string) (sigV4AuthHeader, bool) {
	const prefix = "AWS4-HMAC-SHA256 "
	if !strings.HasPrefix(value, prefix) {
		return sigV4AuthHeader{}, false
	}
	rest := strings.TrimPrefix(value, prefix)
	parts := strings.Split(rest, ", ")
	if len(parts) < 1 {
		return sigV4AuthHeader{}, false
	}
	credPart := strings.TrimPrefix(parts[0], "Credential=")
	scope := strings.Split(credPart, "/")
	if len(scope) < 5 {
		return sigV4AuthHeader{}, false
	}
	return sigV4AuthHeader{
		AccessKey: scope[0],
		Region:    scope[2],
		Service:   scope[3],
	}, true
}

func ossRegionFromHost(host string) string {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(host)), ".")
	if len(parts) >= 4 {
		return parts[1]
	}
	return ""
}
