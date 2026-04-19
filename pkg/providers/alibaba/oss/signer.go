package oss

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
)

var signableQueryKeys = map[string]struct{}{
	"acl":                          {},
	"append":                       {},
	"asyncFetch":                   {},
	"bucketInfo":                   {},
	"callback":                     {},
	"callback-var":                 {},
	"cloudboxes":                   {},
	"cname":                        {},
	"comp":                         {},
	"continuation-token":           {},
	"cors":                         {},
	"delete":                       {},
	"endTime":                      {},
	"encryption":                   {},
	"img":                          {},
	"inventory":                    {},
	"inventoryId":                  {},
	"lifecycle":                    {},
	"location":                     {},
	"logging":                      {},
	"metaQuery":                    {},
	"objectMeta":                   {},
	"partNumber":                   {},
	"policy":                       {},
	"position":                     {},
	"qos":                          {},
	"qosInfo":                      {},
	"referer":                      {},
	"regionList":                   {},
	"replication":                  {},
	"replicationLocation":          {},
	"replicationProgress":          {},
	"requestPayment":               {},
	"resourceGroup":                {},
	"response-cache-control":       {},
	"response-content-disposition": {},
	"response-content-encoding":    {},
	"response-content-language":    {},
	"response-content-type":        {},
	"response-expires":             {},
	"responseHeader":               {},
	"restore":                      {},
	"rtc":                          {},
	"security-token":               {},
	"sequential":                   {},
	"startTime":                    {},
	"stat":                         {},
	"status":                       {},
	"style":                        {},
	"styleName":                    {},
	"symlink":                      {},
	"tagging":                      {},
	"transferAcceleration":         {},
	"udf":                          {},
	"udfApplication":               {},
	"udfApplicationLog":            {},
	"udfId":                        {},
	"udfImage":                     {},
	"udfImageDesc":                 {},
	"udfName":                      {},
	"uploads":                      {},
	"uploadId":                     {},
	"versionId":                    {},
	"versioning":                   {},
	"versions":                     {},
	"vod":                          {},
	"website":                      {},
	"withHashContext":              {},
	"worm":                         {},
	"wormExtend":                   {},
	"wormId":                       {},
	"x-oss-ac-forward-allow":       {},
	"x-oss-ac-source-ip":           {},
	"x-oss-ac-subnet-mask":         {},
	"x-oss-ac-vpc-id":              {},
	"x-oss-async-process":          {},
	"x-oss-enable-md5":             {},
	"x-oss-enable-sha1":            {},
	"x-oss-enable-sha256":          {},
	"x-oss-hash-ctx":               {},
	"x-oss-md5-ctx":                {},
	"x-oss-process":                {},
	"x-oss-request-payer":          {},
	"x-oss-traffic-limit":          {},
}

func Sign(req *http.Request, cred aliauth.Credential, bucket string, now time.Time) error {
	if req == nil {
		return fmt.Errorf("alibaba oss signer: nil request")
	}
	if req.URL == nil {
		return fmt.Errorf("alibaba oss signer: nil request url")
	}
	if err := cred.Validate(); err != nil {
		return err
	}
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	if req.Host == "" {
		req.Host = req.URL.Host
	}
	if req.Header.Get("Date") == "" {
		if now.IsZero() {
			now = time.Now().UTC()
		} else {
			now = now.UTC()
		}
		req.Header.Set("Date", now.Format(http.TimeFormat))
	}
	if cred.SecurityToken != "" {
		req.Header.Set("X-Oss-Security-Token", cred.SecurityToken)
	}

	stringToSign := buildStringToSign(req, bucket)
	signature := signString(stringToSign, cred.AccessKeySecret)
	req.Header.Set("Authorization", fmt.Sprintf("OSS %s:%s", cred.AccessKeyID, signature))
	return nil
}

func buildStringToSign(req *http.Request, bucket string) string {
	canonicalHeaders := canonicalizedOSSHeaders(req.Header)
	return strings.Join([]string{
		req.Method,
		req.Header.Get("Content-MD5"),
		req.Header.Get("Content-Type"),
		req.Header.Get("Date"),
		canonicalHeaders + canonicalizedResource(bucket, req.URL.EscapedPath(), req.URL.Query()),
	}, "\n")
}

func canonicalizedOSSHeaders(headers http.Header) string {
	if len(headers) == 0 {
		return ""
	}
	keys := make([]string, 0, len(headers))
	values := make(map[string]string, len(headers))
	for key, items := range headers {
		lowerKey := strings.ToLower(strings.TrimSpace(key))
		if !strings.HasPrefix(lowerKey, "x-oss-") {
			continue
		}
		keys = append(keys, lowerKey)
		if len(items) > 0 {
			values[lowerKey] = items[0]
		} else {
			values[lowerKey] = ""
		}
	}
	sort.Strings(keys)

	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteByte(':')
		builder.WriteString(values[key])
		builder.WriteByte('\n')
	}
	return builder.String()
}

func canonicalizedResource(bucket, escapedPath string, query url.Values) string {
	bucket = strings.TrimSpace(bucket)
	path := strings.TrimPrefix(escapedPath, "/")
	resource := "/"
	switch {
	case bucket == "":
		resource = "/"
	case path == "":
		resource = "/" + bucket + "/"
	default:
		resource = "/" + bucket + "/" + path
	}

	subresource := canonicalizedSubresource(query)
	if subresource == "" {
		return resource
	}
	return resource + "?" + subresource
}

func canonicalizedSubresource(query url.Values) string {
	if len(query) == 0 {
		return ""
	}
	keys := make([]string, 0, len(query))
	for key := range query {
		if _, ok := signableQueryKeys[key]; ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return ""
	}

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		values := query[key]
		if len(values) == 0 || values[0] == "" {
			parts = append(parts, key)
			continue
		}
		parts = append(parts, key+"="+values[0])
	}
	return strings.Join(parts, "&")
}

func signString(stringToSign, secret string) string {
	mac := hmac.New(sha1.New, []byte(secret))
	_, _ = mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
