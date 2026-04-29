package replay

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/obs"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

func (t *transport) handleOBS(req *http.Request, host, region string, body []byte) (*http.Response, error) {
	switch verifyOBSAuth(req, body) {
	case demoreplay.AuthInvalidAccessKey:
		return obsErrorResponse(req, http.StatusForbidden, "InvalidAccessKeyId",
			"The Access Key Id you provided does not exist in our records."), nil
	case demoreplay.AuthInvalidSignature:
		return obsErrorResponse(req, http.StatusForbidden, "SignatureDoesNotMatch",
			"The request signature we calculated does not match the signature you provided."), nil
	}

	if req.Method != http.MethodGet {
		return obsErrorResponse(req, http.StatusMethodNotAllowed, "MethodNotAllowed",
			"The specified method is not allowed against this resource."), nil
	}

	path := strings.TrimPrefix(req.URL.Path, "/")
	query := req.URL.Query()
	if path == "" {
		return handleListBuckets(req)
	}
	bucket, _, _ := strings.Cut(path, "/")
	return handleListObjects(req, bucket, region, query)
}

func handleListBuckets(req *http.Request) (*http.Response, error) {
	resp := obs.ListBucketsResponse{}
	for _, bucket := range demoOBSBuckets {
		resp.Buckets = append(resp.Buckets, obs.OBSBucket{
			Name:     bucket.Name,
			Location: bucket.Region,
		})
	}
	return xmlResponse(req, http.StatusOK, resp), nil
}

func handleListObjects(req *http.Request, bucketName, region string, query url.Values) (*http.Response, error) {
	bucket, ok := findOBSBucket(bucketName)
	if !ok {
		return obsErrorResponse(req, http.StatusNotFound, "NoSuchBucket",
			"The specified bucket does not exist."), nil
	}
	if region != "" && bucket.Region != region {
		return obsErrorResponse(req, http.StatusMovedPermanently, "PermanentRedirect",
			"The bucket you are attempting to access must be addressed using the specified endpoint."), nil
	}

	maxKeys := demoreplay.ParseInt(query.Get("max-keys"), 1000)
	marker := strings.TrimSpace(query.Get("marker"))
	start := 0
	if marker != "" {
		for i, item := range bucket.Objects {
			if item.Key > marker {
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

	resp := obs.ListObjectsResponse{
		Name:        bucket.Name,
		MaxKeys:     maxKeys,
		Marker:      marker,
		IsTruncated: end < len(bucket.Objects),
	}
	if resp.IsTruncated && len(page) > 0 {
		resp.NextMarker = page[len(page)-1].Key
	}
	for _, item := range page {
		resp.Objects = append(resp.Objects, obs.OBSObject{
			Key:          item.Key,
			Size:         item.Size,
			LastModified: item.LastModified,
			StorageClass: item.StorageClass,
		})
	}
	return xmlResponse(req, http.StatusOK, resp), nil
}

func xmlResponse(req *http.Request, statusCode int, payload any) *http.Response {
	body, _ := xml.Marshal(payload)
	resp := demoreplay.Response(req, statusCode, "application/xml", body)
	resp.Header.Set("X-Obs-Request-Id", "req-replay-obs-"+strconv.Itoa(statusCode))
	return resp
}

type obsErrorPayload struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	RequestID string   `xml:"RequestId"`
	HostID    string   `xml:"HostId"`
}

func obsErrorResponse(req *http.Request, statusCode int, code, message string) *http.Response {
	payload := obsErrorPayload{
		Code:      strings.TrimSpace(code),
		Message:   strings.TrimSpace(message),
		RequestID: fmt.Sprintf("req-replay-obs-%d", statusCode),
		HostID:    "replay",
	}
	body, _ := xml.Marshal(payload)
	resp := demoreplay.Response(req, statusCode, "application/xml", body)
	resp.Header.Set("X-Obs-Request-Id", payload.RequestID)
	return resp
}

func verifyOBSAuth(req *http.Request, body []byte) demoreplay.AuthFailureKind {
	header := strings.TrimSpace(req.Header.Get("Authorization"))
	if header == "" {
		return demoreplay.AuthInvalidSignature
	}
	scheme, ak, _, ok := parseOBSAuth(header)
	if !ok {
		return demoreplay.AuthInvalidSignature
	}
	if ak != demoCredentials.AccessKey {
		return demoreplay.AuthInvalidAccessKey
	}
	timestamp, ok := parseOBSDate(req)
	if !ok {
		return demoreplay.AuthInvalidSignature
	}

	signed, err := obs.Sign(&obs.SignRequest{
		Method:    req.Method,
		Path:      req.URL.Path,
		Query:     req.URL.Query(),
		Headers:   filterOBSSignedHeaders(req.Header),
		Scheme:    scheme,
		AccessKey: demoCredentials.AccessKey,
		SecretKey: demoCredentials.SecretKey,
		Timestamp: timestamp,
	})
	if err != nil {
		return demoreplay.AuthInvalidSignature
	}
	expected := signed.Get("Authorization")
	if demoreplay.SubtleEqual(strings.TrimSpace(expected), header) {
		return demoreplay.AuthOK
	}
	_ = body
	return demoreplay.AuthInvalidSignature
}

func parseOBSAuth(value string) (string, string, string, bool) {
	parts := strings.SplitN(value, " ", 2)
	if len(parts) != 2 {
		return "", "", "", false
	}
	scheme := strings.TrimSpace(parts[0])
	creds := strings.SplitN(strings.TrimSpace(parts[1]), ":", 2)
	if len(creds) != 2 {
		return "", "", "", false
	}
	return scheme, strings.TrimSpace(creds[0]), strings.TrimSpace(creds[1]), true
}

func parseOBSDate(req *http.Request) (time.Time, bool) {
	for _, name := range []string{"X-Obs-Date", "Date"} {
		value := strings.TrimSpace(req.Header.Get(name))
		if value == "" {
			continue
		}
		if ts, err := http.ParseTime(value); err == nil {
			return ts.UTC(), true
		}
	}
	return time.Time{}, false
}

func filterOBSSignedHeaders(headers http.Header) http.Header {
	out := http.Header{}
	for key, values := range headers {
		lower := strings.ToLower(strings.TrimSpace(key))
		switch lower {
		case "authorization", "host":
			continue
		}
		out[key] = append([]string(nil), values...)
	}
	return out
}
