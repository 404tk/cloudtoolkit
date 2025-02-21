package sls

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"
)

const HeaderSLSPrefix1 = "x-log-"
const HeaderSLSPrefix2 = "x-acs-"

func (client *Client) signRequest(req *request, payload []byte) {
	if _, ok := req.headers["Authorization"]; ok {
		return
	}

	contentMd5 := ""
	contentType := req.headers["Content-Type"]
	if req.payload != nil {
		hasher := md5.New()
		hasher.Write(payload)
		contentMd5 = strings.ToUpper(hex.EncodeToString(hasher.Sum(nil)))
		req.headers["Content-MD5"] = contentMd5

	}
	date := req.headers["Date"]
	canonicalizedHeader := canonicalizeHeader(req.headers)

	canonicalizedResource := canonicalizeResource(req)

	signString := req.method + "\n" + contentMd5 + "\n" + contentType + "\n" + date + "\n" + canonicalizedHeader + "\n" + canonicalizedResource

	signature := CreateSignature(signString, client.accessKeySecret)
	req.headers["Authorization"] = "LOG " + client.accessKeyId + ":" + signature
}

func canonicalizeResource(req *request) string {
	canonicalizedResource := req.path
	var paramNames []string
	if len(req.params) > 0 {
		for k := range req.params {
			paramNames = append(paramNames, k)
		}
		sort.Strings(paramNames)

		var query []string
		for _, k := range paramNames {
			query = append(query, url.QueryEscape(k)+"="+url.QueryEscape(req.params[k]))
		}
		canonicalizedResource = canonicalizedResource + "?" + strings.Join(query, "&")
	}
	return canonicalizedResource
}

// Have to break the abstraction to append keys with lower case.
func canonicalizeHeader(headers map[string]string) string {
	var canonicalizedHeaders []string

	for k := range headers {
		if lower := strings.ToLower(k); strings.HasPrefix(lower, HeaderSLSPrefix1) || strings.HasPrefix(lower, HeaderSLSPrefix2) {
			canonicalizedHeaders = append(canonicalizedHeaders, lower)
		}
	}

	sort.Strings(canonicalizedHeaders)

	var headersWithValue []string

	for _, k := range canonicalizedHeaders {
		headersWithValue = append(headersWithValue, k+":"+headers[k])
	}
	return strings.Join(headersWithValue, "\n")
}

// CreateSignature creates signature for string following Aliyun rules
func CreateSignature(stringToSignature, accessKeySecret string) string {
	// Crypto by HMAC-SHA1
	hmacSha1 := hmac.New(sha1.New, []byte(accessKeySecret))
	hmacSha1.Write([]byte(stringToSignature))
	sign := hmacSha1.Sum(nil)

	// Encode to Base64
	base64Sign := base64.StdEncoding.EncodeToString(sign)

	return base64Sign
}
