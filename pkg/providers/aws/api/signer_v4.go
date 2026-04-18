package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/auth"
)

type SignInput struct {
	Method      string
	Service     string
	Region      string
	Host        string
	Path        string
	Query       url.Values
	ContentType string
	Payload     []byte
	Timestamp   time.Time
	Headers     http.Header
}

type Signature struct {
	Authorization    string
	SignedHeaders    string
	CredentialScope  string
	CanonicalRequest string
	StringToSign     string
	AmzDate          string
	PayloadHash      string
}

type SigV4Signer struct{}

func (s SigV4Signer) Sign(credential auth.Credential, input SignInput) (Signature, error) {
	if err := credential.Validate(); err != nil {
		return Signature{}, err
	}
	service := strings.TrimSpace(strings.ToLower(input.Service))
	if service == "" {
		return Signature{}, fmt.Errorf("aws signer: empty service")
	}
	region := strings.TrimSpace(input.Region)
	if region == "" {
		return Signature{}, fmt.Errorf("aws signer: empty region")
	}
	host := strings.TrimSpace(input.Host)
	if host == "" {
		return Signature{}, fmt.Errorf("aws signer: empty host")
	}

	method := strings.ToUpper(strings.TrimSpace(input.Method))
	if method == "" {
		method = http.MethodPost
	}
	amzDate := input.Timestamp.UTC().Format("20060102T150405Z")
	shortDate := input.Timestamp.UTC().Format("20060102")
	payloadHash := hashSHA256Hex(input.Payload)
	canonicalHeaders, signedHeaders := buildCanonicalHeaders(host, input.ContentType, amzDate, credential.SessionToken, len(input.Payload), input.Headers)
	credentialScope := shortDate + "/" + region + "/" + service + "/aws4_request"
	canonicalRequest := strings.Join([]string{
		method,
		canonicalURI(input.Path),
		canonicalQuery(input.Query),
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		hashSHA256Hex([]byte(canonicalRequest)),
	}, "\n")
	signature := hex.EncodeToString(signV4(credential.SecretAccessKey, shortDate, region, service, stringToSign))

	return Signature{
		Authorization: fmt.Sprintf(
			"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
			credential.AccessKeyID,
			credentialScope,
			signedHeaders,
			signature,
		),
		SignedHeaders:    signedHeaders,
		CredentialScope:  credentialScope,
		CanonicalRequest: canonicalRequest,
		StringToSign:     stringToSign,
		AmzDate:          amzDate,
		PayloadHash:      payloadHash,
	}, nil
}

func buildCanonicalHeaders(host, contentType, amzDate, sessionToken string, payloadLength int, extra http.Header) (string, string) {
	headers := map[string]string{
		"host":       strings.TrimSpace(host),
		"x-amz-date": amzDate,
	}
	if payloadLength > 0 {
		headers["content-length"] = strconv.Itoa(payloadLength)
	}
	if value := strings.TrimSpace(contentType); value != "" {
		headers["content-type"] = value
	}
	if value := strings.TrimSpace(sessionToken); value != "" {
		headers["x-amz-security-token"] = value
	}
	for key, values := range extra {
		name := strings.ToLower(strings.TrimSpace(key))
		if name == "" {
			continue
		}
		headers[name] = normalizeHeaderValue(strings.Join(values, ","))
	}

	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	sort.Strings(names)

	var canonical strings.Builder
	for _, name := range names {
		canonical.WriteString(name)
		canonical.WriteByte(':')
		canonical.WriteString(normalizeHeaderValue(headers[name]))
		canonical.WriteByte('\n')
	}
	return canonical.String(), strings.Join(names, ";")
}

func canonicalURI(path string) string {
	path = ensureLeadingSlash(path)
	if path == "/" {
		return path
	}
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		segments[i] = awsPercentEncode(segment)
	}
	return strings.Join(segments, "/")
}

func canonicalQuery(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	type pair struct {
		key   string
		value string
	}
	pairs := make([]pair, 0, len(values))
	for key, rawValues := range values {
		encodedKey := awsPercentEncode(key)
		if len(rawValues) == 0 {
			pairs = append(pairs, pair{key: encodedKey})
			continue
		}
		sortedValues := append([]string(nil), rawValues...)
		sort.Strings(sortedValues)
		for _, value := range sortedValues {
			pairs = append(pairs, pair{key: encodedKey, value: awsPercentEncode(value)})
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].key != pairs[j].key {
			return pairs[i].key < pairs[j].key
		}
		return pairs[i].value < pairs[j].value
	})

	var canonical strings.Builder
	for i, item := range pairs {
		if i > 0 {
			canonical.WriteByte('&')
		}
		canonical.WriteString(item.key)
		canonical.WriteByte('=')
		canonical.WriteString(item.value)
	}
	return canonical.String()
}

func normalizeHeaderValue(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func hashSHA256Hex(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func signV4(secretAccessKey, shortDate, region, service, stringToSign string) []byte {
	dateKey := hmacSHA256([]byte("AWS4"+secretAccessKey), shortDate)
	regionKey := hmacSHA256(dateKey, region)
	serviceKey := hmacSHA256(regionKey, service)
	signingKey := hmacSHA256(serviceKey, "aws4_request")
	return hmacSHA256(signingKey, stringToSign)
}

func hmacSHA256(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

func awsPercentEncode(value string) string {
	escaped := url.QueryEscape(value)
	escaped = strings.ReplaceAll(escaped, "+", "%20")
	escaped = strings.ReplaceAll(escaped, "*", "%2A")
	escaped = strings.ReplaceAll(escaped, "%7E", "~")
	return escaped
}
