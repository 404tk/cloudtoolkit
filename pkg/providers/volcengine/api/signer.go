package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/internal/httpclient"
)

const (
	Algorithm            = "HMAC-SHA256"
	DateFormat           = "20060102T150405Z"
	HeaderAuthorization  = "Authorization"
	HeaderXDate          = "X-Date"
	HeaderXContentSHA256 = "X-Content-Sha256"
	HeaderXSecurityToken = "X-Security-Token"
)

type SignInput struct {
	Method       string
	Host         string
	Path         string
	Query        url.Values
	Body         []byte
	ContentType  string
	Service      string
	Region       string
	AccessKey    string
	SecretKey    string
	SessionToken string
	Headers      http.Header
	Timestamp    time.Time
}

type Signature struct {
	Authorization    string
	SignedHeaders    string
	CredentialScope  string
	CanonicalRequest string
	StringToSign     string
	XDate            string
	XContentSHA256   string
}

func Sign(input SignInput) (Signature, error) {
	method := strings.ToUpper(strings.TrimSpace(input.Method))
	if method == "" {
		method = http.MethodGet
	}
	host := normalizeHost(input.Host)
	if host == "" {
		return Signature{}, fmt.Errorf("volcengine signer: empty host")
	}
	service := strings.ToLower(strings.TrimSpace(input.Service))
	if service == "" {
		return Signature{}, fmt.Errorf("volcengine signer: empty service")
	}
	region := strings.TrimSpace(input.Region)
	if region == "" {
		return Signature{}, fmt.Errorf("volcengine signer: empty region")
	}
	accessKey := strings.TrimSpace(input.AccessKey)
	if accessKey == "" {
		return Signature{}, fmt.Errorf("volcengine signer: empty access key")
	}
	secretKey := strings.TrimSpace(input.SecretKey)
	if secretKey == "" {
		return Signature{}, fmt.Errorf("volcengine signer: empty secret key")
	}

	timestamp := input.Timestamp.UTC()
	if input.Timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	xDate := timestamp.Format(DateFormat)
	payloadHash := hashSHA256Hex(input.Body)

	headers := canonicalSignHeaders(input.Headers)
	headers["host"] = host
	headers["x-date"] = xDate
	headers["x-content-sha256"] = payloadHash
	if contentType := strings.TrimSpace(input.ContentType); contentType != "" {
		headers["content-type"] = contentType
	}
	if token := strings.TrimSpace(input.SessionToken); token != "" {
		headers["x-security-token"] = token
	}

	signedHeaders := signedHeaderNames(headers)
	canonicalRequest := strings.Join([]string{
		method,
		canonicalURI(input.Path),
		canonicalQueryString(input.Query),
		canonicalHeaders(headers, signedHeaders),
		strings.Join(signedHeaders, ";"),
		payloadHash,
	}, "\n")

	shortDate := xDate[:8]
	credentialScope := shortDate + "/" + region + "/" + service + "/request"
	stringToSign := strings.Join([]string{
		Algorithm,
		xDate,
		credentialScope,
		hashSHA256Hex([]byte(canonicalRequest)),
	}, "\n")

	signature := hex.EncodeToString(signVolc(secretKey, shortDate, region, service, stringToSign))
	signedHeaderLine := strings.Join(signedHeaders, ";")
	return Signature{
		Authorization:    fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s", Algorithm, accessKey, credentialScope, signedHeaderLine, signature),
		SignedHeaders:    signedHeaderLine,
		CredentialScope:  credentialScope,
		CanonicalRequest: canonicalRequest,
		StringToSign:     stringToSign,
		XDate:            xDate,
		XContentSHA256:   payloadHash,
	}, nil
}

func canonicalSignHeaders(headers http.Header) map[string]string {
	normalized := make(map[string]string, len(headers))
	for key, values := range headers {
		name := strings.ToLower(strings.TrimSpace(key))
		if name == "" || name == strings.ToLower(HeaderAuthorization) {
			continue
		}
		normalized[name] = strings.TrimSpace(strings.Join(values, ","))
	}
	return normalized
}

func signedHeaderNames(headers map[string]string) []string {
	names := make([]string, 0, len(headers))
	for name := range headers {
		switch {
		case name == "host":
			names = append(names, name)
		case name == "content-type":
			names = append(names, name)
		case name == "content-md5":
			names = append(names, name)
		case strings.HasPrefix(name, "x-"):
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func canonicalHeaders(headers map[string]string, signedHeaders []string) string {
	lines := make([]string, 0, len(signedHeaders))
	for _, name := range signedHeaders {
		lines = append(lines, name+":"+strings.TrimSpace(headers[name]))
	}
	return strings.Join(lines, "\n") + "\n"
}

func canonicalQueryString(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	return strings.ReplaceAll(values.Encode(), "+", "%20")
}

func canonicalURI(path string) string {
	path = httpclient.EnsureLeadingSlash(path)
	if path == "/" {
		return "/"
	}
	parts := strings.Split(path, "/")
	for i, part := range parts {
		parts[i] = percentEncodeRFC3986(part)
	}
	return strings.Join(parts, "/")
}

func percentEncodeRFC3986(value string) string {
	if value == "" {
		return ""
	}
	var builder strings.Builder
	builder.Grow(len(value) * 3)
	for i := 0; i < len(value); i++ {
		c := value[i]
		if isUnreserved(c) {
			builder.WriteByte(c)
			continue
		}
		builder.WriteByte('%')
		builder.WriteByte(upperhex[c>>4])
		builder.WriteByte(upperhex[c&15])
	}
	return builder.String()
}

var upperhex = "0123456789ABCDEF"

func isUnreserved(c byte) bool {
	return ('A' <= c && c <= 'Z') ||
		('a' <= c && c <= 'z') ||
		('0' <= c && c <= '9') ||
		c == '-' || c == '_' || c == '.' || c == '~'
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if strings.Contains(host, "://") {
		if u, err := url.Parse(host); err == nil && u.Host != "" {
			host = u.Host
		}
	}
	if strings.HasSuffix(host, ":80") {
		return strings.TrimSuffix(host, ":80")
	}
	if strings.HasSuffix(host, ":443") {
		return strings.TrimSuffix(host, ":443")
	}
	return host
}

func hashSHA256Hex(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func signVolc(secretKey, shortDate, region, service, stringToSign string) []byte {
	dateKey := hmacSHA256([]byte(secretKey), shortDate)
	regionKey := hmacSHA256(dateKey, region)
	serviceKey := hmacSHA256(regionKey, service)
	signingKey := hmacSHA256(serviceKey, "request")
	return hmacSHA256(signingKey, stringToSign)
}

func hmacSHA256(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}
