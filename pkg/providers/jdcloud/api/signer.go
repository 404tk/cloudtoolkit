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
)

const (
	Algorithm           = "JDCLOUD2-HMAC-SHA256"
	SigningTerm         = "jdcloud2_request"
	TimeFormat          = "20060102T150405Z"
	HeaderAuthorization = "Authorization"
	HeaderXJdcloudDate  = "X-Jdcloud-Date"
	HeaderXJdcloudNonce = "X-Jdcloud-Nonce"
	HeaderXJdcloudToken = "X-Jdcloud-Security-Token"
	emptyBodySHA256Hex  = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

var ignoredHeaderNames = map[string]struct{}{
	"authorization":        {},
	"user-agent":           {},
	"x-jdcloud-request-id": {},
}

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
	Nonce        string
	Timestamp    time.Time
	Headers      http.Header
}

type Signature struct {
	Authorization    string
	SignedHeaders    string
	CredentialScope  string
	CanonicalRequest string
	StringToSign     string
	XJdcloudDate     string
	XJdcloudNonce    string
	BodyDigest       string
}

func Sign(input SignInput) (Signature, error) {
	method := strings.ToUpper(strings.TrimSpace(input.Method))
	if method == "" {
		method = http.MethodGet
	}
	host := normalizeHost(input.Host)
	if host == "" {
		return Signature{}, fmt.Errorf("jdcloud signer: empty host")
	}
	service := strings.ToLower(strings.TrimSpace(input.Service))
	if service == "" {
		return Signature{}, fmt.Errorf("jdcloud signer: empty service")
	}
	region := ResolveSigningRegion(input.Region)
	accessKey := strings.TrimSpace(input.AccessKey)
	if accessKey == "" {
		return Signature{}, fmt.Errorf("jdcloud signer: empty access key")
	}
	secretKey := strings.TrimSpace(input.SecretKey)
	if secretKey == "" {
		return Signature{}, fmt.Errorf("jdcloud signer: empty secret key")
	}
	nonce := strings.TrimSpace(input.Nonce)
	if nonce == "" {
		return Signature{}, fmt.Errorf("jdcloud signer: empty nonce")
	}

	timestamp := input.Timestamp.UTC()
	if input.Timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	xDate := timestamp.Format(TimeFormat)
	payloadHash := hashSHA256Hex(input.Body)

	headers := canonicalHeadersMap(input.Headers)
	headers["host"] = host
	headers["x-jdcloud-date"] = xDate
	headers["x-jdcloud-nonce"] = nonce
	if contentType := strings.TrimSpace(input.ContentType); contentType != "" {
		headers["content-type"] = contentType
	}
	if token := strings.TrimSpace(input.SessionToken); token != "" {
		headers["x-jdcloud-security-token"] = token
	}

	names := sortedHeaderNames(headers)
	signedHeaders := strings.Join(names, ";")
	canonicalRequest := strings.Join([]string{
		method,
		canonicalURI(input.Path),
		canonicalQuery(input.Query),
		canonicalHeaderBlock(headers, names),
		signedHeaders,
		payloadHash,
	}, "\n")

	shortDate := xDate[:8]
	credentialScope := shortDate + "/" + region + "/" + service + "/" + SigningTerm
	stringToSign := strings.Join([]string{
		Algorithm,
		xDate,
		credentialScope,
		hashSHA256Hex([]byte(canonicalRequest)),
	}, "\n")

	signature := hex.EncodeToString(signJDCLOUD2(secretKey, shortDate, region, service, stringToSign))
	return Signature{
		Authorization: fmt.Sprintf(
			"%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
			Algorithm,
			accessKey,
			credentialScope,
			signedHeaders,
			signature,
		),
		SignedHeaders:    signedHeaders,
		CredentialScope:  credentialScope,
		CanonicalRequest: canonicalRequest,
		StringToSign:     stringToSign,
		XJdcloudDate:     xDate,
		XJdcloudNonce:    nonce,
		BodyDigest:       payloadHash,
	}, nil
}

func canonicalHeadersMap(headers http.Header) map[string]string {
	if len(headers) == 0 {
		return map[string]string{}
	}
	normalized := make(map[string]string, len(headers))
	for key, values := range headers {
		name := strings.ToLower(strings.TrimSpace(key))
		if name == "" {
			continue
		}
		if _, ignored := ignoredHeaderNames[name]; ignored {
			continue
		}
		normalized[name] = strings.TrimSpace(strings.Join(values, ","))
	}
	return normalized
}

func sortedHeaderNames(headers map[string]string) []string {
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func canonicalHeaderBlock(headers map[string]string, names []string) string {
	lines := make([]string, 0, len(names))
	for _, name := range names {
		lines = append(lines, name+":"+strings.TrimSpace(headers[name]))
	}
	return strings.Join(lines, "\n") + "\n"
}

func canonicalQuery(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	return strings.ReplaceAll(values.Encode(), "+", "%20")
}

func canonicalURI(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	u, err := url.Parse(path)
	if err != nil || u.Path == "" {
		return EscapePath(path)
	}
	return EscapePath(u.Path)
}

// EscapePath percent-encodes every byte that is not in the RFC 3986 unreserved
// set (A-Z / a-z / 0-9 / '-' / '.' / '_' / '~') or '/'. This matches JDCloud's
// official SDK (`core.EscapePath(path, false)`) so the canonical URI fed into
// the signer is byte-identical to what the service recomputes, including
// reserved characters like ':' that Go's default EscapedPath leaves alone.
func EscapePath(path string) string {
	if path == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(path))
	for i := 0; i < len(path); i++ {
		c := path[i]
		if isUnreservedPathByte(c) || c == '/' {
			b.WriteByte(c)
			continue
		}
		const hex = "0123456789ABCDEF"
		b.WriteByte('%')
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&0x0f])
	}
	return b.String()
}

func isUnreservedPathByte(c byte) bool {
	switch {
	case c >= 'A' && c <= 'Z':
		return true
	case c >= 'a' && c <= 'z':
		return true
	case c >= '0' && c <= '9':
		return true
	case c == '-' || c == '.' || c == '_' || c == '~':
		return true
	}
	return false
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

func hashSHA256Hex(payload []byte) string {
	if len(payload) == 0 {
		return emptyBodySHA256Hex
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func signJDCLOUD2(secretKey, shortDate, region, service, stringToSign string) []byte {
	dateKey := hmacSHA256([]byte("JDCLOUD2"+secretKey), shortDate)
	regionKey := hmacSHA256(dateKey, region)
	serviceKey := hmacSHA256(regionKey, service)
	signingKey := hmacSHA256(serviceKey, SigningTerm)
	return hmacSHA256(signingKey, stringToSign)
}

func hmacSHA256(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}
