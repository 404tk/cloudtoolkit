package obs

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
)

const (
	authScheme    = "OBS"
	authSchemeV2  = "AWS"
	dateHeader    = "Date"
	authHeader    = "Authorization"
	obsHeaderPref = "x-obs-"
)

var allowedResourceParameters = map[string]struct{}{
	"acl":                          {},
	"backtosource":                 {},
	"bucketstatus":                 {},
	"cdnnotifyconfiguration":       {},
	"cors":                         {},
	"customdomain":                 {},
	"delete":                       {},
	"deletebucket":                 {},
	"directcoldaccess":             {},
	"encryption":                   {},
	"ignore-sign-in-query":         {},
	"inventory":                    {},
	"length":                       {},
	"lifecycle":                    {},
	"location":                     {},
	"logging":                      {},
	"metadata":                     {},
	"mirrorbacktosource":           {},
	"modify":                       {},
	"name":                         {},
	"notification":                 {},
	"object-lock":                  {},
	"policy":                       {},
	"policystatus":                 {},
	"position":                     {},
	"publicaccessblock":            {},
	"quota":                        {},
	"rename":                       {},
	"replication":                  {},
	"requestpayment":               {},
	"response-cache-control":       {},
	"response-content-disposition": {},
	"response-content-encoding":    {},
	"response-content-language":    {},
	"response-content-type":        {},
	"response-expires":             {},
	"restore":                      {},
	"retention":                    {},
	"storageclass":                 {},
	"storageinfo":                  {},
	"storagepolicy":                {},
	"tagging":                      {},
	"torrent":                      {},
	"truncate":                     {},
	"uploads":                      {},
	"versionid":                    {},
	"versioning":                   {},
	"versions":                     {},
	"website":                      {},
	"x-image-process":              {},
	"x-image-save-bucket":          {},
	"x-image-save-object":          {},
	"x-obs-accesslabel":            {},
	"x-obs-security-token":         {},
	"x-oss-process":                {},
}

type SignRequest struct {
	Method    string
	Path      string
	Query     url.Values
	Headers   http.Header
	Scheme    string
	AccessKey string
	SecretKey string
	Timestamp time.Time
}

func Sign(req *SignRequest) (http.Header, error) {
	if req == nil {
		return nil, fmt.Errorf("huawei obs signer: nil request")
	}
	if strings.TrimSpace(req.AccessKey) == "" {
		return nil, fmt.Errorf("huawei obs signer: empty access key")
	}
	if strings.TrimSpace(req.SecretKey) == "" {
		return nil, fmt.Errorf("huawei obs signer: empty secret key")
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}

	headers := cloneHeaders(req.Headers)
	dateValue := firstHeaderValue(headers, dateHeader)
	if dateValue == "" && firstHeaderValue(headers, obsHeaderPref+"date") == "" {
		ts := req.Timestamp.UTC()
		if req.Timestamp.IsZero() {
			ts = time.Now().UTC()
		}
		dateValue = ts.Format(http.TimeFormat)
		headers.Set(dateHeader, dateValue)
	}

	stringToSign := buildStringToSign(method, req.Path, req.Query, headers)
	signature := signString(stringToSign, req.SecretKey)
	scheme := strings.TrimSpace(req.Scheme)
	if scheme == "" {
		scheme = authScheme
	}
	headers.Set(authHeader, fmt.Sprintf("%s %s:%s", scheme, req.AccessKey, signature))
	return headers, nil
}

func buildStringToSign(method, path string, query url.Values, headers http.Header) string {
	normalized := normalizeHeaders(headers)
	dateValue := normalized["date"]
	if normalized[obsHeaderPref+"date"] != "" {
		dateValue = ""
	}

	var builder strings.Builder
	builder.WriteString(method)
	builder.WriteByte('\n')
	builder.WriteString(normalized["content-md5"])
	builder.WriteByte('\n')
	builder.WriteString(normalized["content-type"])
	builder.WriteByte('\n')
	builder.WriteString(dateValue)
	builder.WriteByte('\n')

	if canonical := canonicalOBSHeaders(normalized); canonical != "" {
		builder.WriteString(canonical)
	}
	builder.WriteString(canonicalResource(path, query))
	return builder.String()
}

func canonicalOBSHeaders(headers map[string]string) string {
	keys := make([]string, 0)
	for key := range headers {
		if strings.HasPrefix(key, obsHeaderPref) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return ""
	}

	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteByte(':')
		builder.WriteString(strings.TrimSpace(headers[key]))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func canonicalResource(path string, query url.Values) string {
	resource := ensureLeadingSlash(path)
	if len(query) == 0 {
		return resource
	}

	normalizedQuery := make(map[string][]string, len(query))
	for key := range query {
		lower := strings.ToLower(strings.TrimSpace(key))
		if lower == "" {
			continue
		}
		if _, ok := allowedResourceParameters[lower]; ok || strings.HasPrefix(lower, obsHeaderPref) {
			normalizedQuery[lower] = append(normalizedQuery[lower], query[key]...)
		}
	}
	keys := make([]string, 0, len(normalizedQuery))
	for key := range normalizedQuery {
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return resource
	}

	sort.Strings(keys)
	var pairs []string
	for _, key := range keys {
		values := append([]string(nil), normalizedQuery[key]...)
		if len(values) == 0 {
			pairs = append(pairs, key)
			continue
		}
		sort.Strings(values)
		for _, value := range values {
			if strings.TrimSpace(value) == "" {
				pairs = append(pairs, key)
				continue
			}
			pairs = append(pairs, key+"="+value)
		}
	}
	return resource + "?" + strings.Join(pairs, "&")
}

func signString(stringToSign, secretKey string) string {
	mac := hmac.New(sha1.New, []byte(secretKey))
	_, _ = mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func normalizeHeaders(headers http.Header) map[string]string {
	normalized := make(map[string]string, len(headers))
	for key, values := range headers {
		name := strings.ToLower(strings.TrimSpace(key))
		if name == "" {
			continue
		}
		normalized[name] = strings.Join(values, ",")
	}
	return normalized
}

func cloneHeaders(headers http.Header) http.Header {
	if headers == nil {
		return http.Header{}
	}
	return headers.Clone()
}

func firstHeaderValue(headers http.Header, name string) string {
	for key, values := range headers {
		if strings.EqualFold(key, name) && len(values) > 0 {
			return strings.TrimSpace(values[0])
		}
	}
	return ""
}

func ensureLeadingSlash(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if strings.HasPrefix(path, "/") {
		return path
	}
	return "/" + path
}
