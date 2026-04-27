package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/internal/httpclient"
)

const (
	BasicDateFormat     = "20060102T150405Z"
	Algorithm           = "SDK-HMAC-SHA256"
	HeaderXDate         = "X-Sdk-Date"
	HeaderHost          = "host"
	HeaderAuthorization = "Authorization"
	HeaderContentSha256 = "X-Sdk-Content-Sha256"
)

type SignRequest struct {
	Method    string
	Host      string
	Path      string
	Query     url.Values
	Headers   map[string]string
	Body      []byte
	AccessKey string
	SecretKey string
	Timestamp time.Time
}

func Sign(req *SignRequest) (map[string]string, error) {
	if req == nil {
		return nil, fmt.Errorf("huawei signer: nil request")
	}
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}
	if normalizeHost(req.Host) == "" {
		return nil, fmt.Errorf("huawei signer: empty host")
	}
	if strings.TrimSpace(req.AccessKey) == "" {
		return nil, fmt.Errorf("huawei signer: empty access key")
	}
	if strings.TrimSpace(req.SecretKey) == "" {
		return nil, fmt.Errorf("huawei signer: empty secret key")
	}

	timestamp := req.Timestamp.UTC()
	if req.Timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	xDate := timestamp.Format(BasicDateFormat)
	payloadHash := hexEncodeSHA256Hash(req.Body)

	signingHeaders := normalizeHeaders(req.Headers)
	signingHeaders[strings.ToLower(HeaderXDate)] = xDate

	signedHeaders := signedHeaderNames(signingHeaders)
	canonicalRequest := strings.Join([]string{
		method,
		canonicalURI(req.Path),
		canonicalQueryString(req.Query),
		canonicalHeaders(signingHeaders, signedHeaders),
		strings.Join(signedHeaders, ";"),
		payloadHash,
	}, "\n")
	stringToSign := strings.Join([]string{
		Algorithm,
		xDate,
		hexEncodeSHA256Hash([]byte(canonicalRequest)),
	}, "\n")
	signature := signStringToSign(stringToSign, req.SecretKey)

	return map[string]string{
		HeaderAuthorization: AuthHeaderValue(signature, req.AccessKey, signedHeaders),
		HeaderXDate:         xDate,
	}, nil
}

func AuthHeaderValue(signature, accessKey string, signedHeaders []string) string {
	return fmt.Sprintf("%s Access=%s, SignedHeaders=%s, Signature=%s",
		Algorithm, accessKey, strings.Join(signedHeaders, ";"), signature)
}

func normalizeHeaders(headers map[string]string) map[string]string {
	normalized := make(map[string]string, len(headers))
	for key, value := range headers {
		name := strings.ToLower(strings.TrimSpace(key))
		if name == "" {
			continue
		}
		normalized[name] = value
	}
	return normalized
}

func signedHeaderNames(headers map[string]string) []string {
	names := make([]string, 0, len(headers))
	for key := range headers {
		switch {
		case strings.EqualFold(key, HeaderAuthorization):
			continue
		case strings.EqualFold(key, HeaderHost):
			continue
		case strings.HasPrefix(strings.ToLower(key), "content-type"):
			continue
		case strings.Contains(key, "_"):
			continue
		default:
			names = append(names, strings.ToLower(key))
		}
	}
	sort.Strings(names)
	return names
}

func canonicalURI(path string) string {
	parts := strings.Split(httpclient.EnsureLeadingSlash(path), "/")
	escaped := make([]string, 0, len(parts))
	for _, part := range parts {
		escaped = append(escaped, escape(part))
	}
	uri := strings.Join(escaped, "/")
	if uri == "" {
		return "/"
	}
	if uri[len(uri)-1] != '/' {
		uri += "/"
	}
	return uri
}

func canonicalQueryString(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	pairs := make([]string, 0)
	for _, key := range keys {
		encodedKey := escape(key)
		sortedValues := append([]string(nil), values[key]...)
		if len(sortedValues) == 0 {
			pairs = append(pairs, encodedKey+"=")
			continue
		}
		sort.Strings(sortedValues)
		for _, value := range sortedValues {
			pairs = append(pairs, encodedKey+"="+escape(value))
		}
	}
	return strings.Join(pairs, "&")
}

func canonicalHeaders(headers map[string]string, signedHeaders []string) string {
	lines := make([]string, 0, len(signedHeaders))
	for _, key := range signedHeaders {
		lines = append(lines, key+":"+strings.TrimSpace(headers[key]))
	}
	return strings.Join(lines, "\n") + "\n"
}

func hexEncodeSHA256Hash(body []byte) string {
	hash := sha256.Sum256(body)
	return fmt.Sprintf("%x", hash[:])
}

func signStringToSign(stringToSign, secretKey string) string {
	mac := hmac.New(sha256.New, []byte(secretKey))
	_, _ = mac.Write([]byte(stringToSign))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if strings.Contains(host, "://") {
		if u, err := url.Parse(host); err == nil && u.Host != "" {
			return u.Host
		}
	}
	if strings.Contains(host, "/") {
		if u, err := url.Parse("https://" + host); err == nil && u.Host != "" {
			return u.Host
		}
	}
	return host
}

func shouldEscape(c byte) bool {
	if 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' || '0' <= c && c <= '9' || c == '_' || c == '-' || c == '~' || c == '.' {
		return false
	}
	return true
}

func escape(s string) string {
	hexCount := 0
	for i := 0; i < len(s); i++ {
		if shouldEscape(s[i]) {
			hexCount++
		}
	}
	if hexCount == 0 {
		return s
	}

	buf := make([]byte, len(s)+2*hexCount)
	j := 0
	for i := 0; i < len(s); i++ {
		if shouldEscape(s[i]) {
			buf[j] = '%'
			buf[j+1] = "0123456789ABCDEF"[s[i]>>4]
			buf[j+2] = "0123456789ABCDEF"[s[i]&15]
			j += 3
			continue
		}
		buf[j] = s[i]
		j++
	}
	return string(buf)
}
