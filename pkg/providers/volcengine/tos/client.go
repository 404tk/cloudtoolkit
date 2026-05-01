package tos

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/internal/httpclient"
	volcapi "github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/auth"
)

const (
	signAlgorithm        = "TOS4-HMAC-SHA256"
	signDateFormat       = "20060102T150405Z"
	headerAuthorization  = "Authorization"
	headerDate           = "Date"
	headerSecurityToken  = "X-Tos-Security-Token"
	headerXDate          = "X-Tos-Date"
	headerXContentSHA256 = "X-Tos-Content-Sha256"
)

type Option func(*Client)

type Client struct {
	credential  auth.Credential
	httpClient  *http.Client
	retryPolicy volcapi.RetryPolicy
	now         func() time.Time
	baseURL     *url.URL
}

type request struct {
	Method  string
	Host    string
	Path    string
	Query   url.Values
	Body    []byte
	Headers http.Header
}

func NewClient(cred auth.Credential, opts ...Option) *Client {
	client := &Client{
		credential:  cred,
		httpClient:  volcapi.NewHTTPClient(),
		retryPolicy: volcapi.DefaultRetryPolicy(),
		now:         time.Now,
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

func WithRetryPolicy(p volcapi.RetryPolicy) Option {
	return func(c *Client) {
		c.retryPolicy = p
	}
}

func WithClock(now func() time.Time) Option {
	return func(c *Client) {
		if now != nil {
			c.now = now
		}
	}
}

func WithBaseURL(rawURL string) Option {
	return func(c *Client) {
		if rawURL == "" {
			return
		}
		if u, err := url.Parse(rawURL); err == nil {
			c.baseURL = u
		}
	}
}

func (c *Client) ListBuckets(ctx context.Context, region string) (ListBucketsOutput, error) {
	var out ListBucketsOutput
	err := c.doJSON(ctx, request{
		Method: http.MethodGet,
		Host:   serviceHost(region),
		Path:   "/",
	}, &out)
	return out, err
}

func (c *Client) ListObjectsV2(ctx context.Context, bucket, region, token string, maxKeys int) (ListObjectsV2Output, error) {
	query := url.Values{}
	query.Set("list-type", "2")
	if maxKeys > 0 {
		query.Set("max-keys", fmt.Sprintf("%d", maxKeys))
	}
	if token = strings.TrimSpace(token); token != "" {
		query.Set("continuation-token", token)
	}

	var out ListObjectsV2Output
	err := c.doJSON(ctx, request{
		Method: http.MethodGet,
		Host:   bucketHost(bucket, region),
		Path:   "/",
		Query:  query,
	}, &out)
	return out, err
}

func (c *Client) doJSON(ctx context.Context, req request, out any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.credential.Validate(); err != nil {
		return err
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}
	signHost := normalizeHost(req.Host)
	if signHost == "" {
		return fmt.Errorf("volcengine tos: empty host")
	}
	body := append([]byte(nil), req.Body...)
	headers, err := c.signHeaders(method, signHost, req.Path, req.Query, body, req.Headers)
	if err != nil {
		return err
	}
	headers.Set("Accept", "application/json")
	for key, values := range req.Headers {
		if strings.EqualFold(key, "host") {
			continue
		}
		for _, v := range values {
			headers.Add(key, v)
		}
	}

	requestURL, err := c.requestURL(signHost, req.Path, req.Query)
	if err != nil {
		return err
	}
	httpResp, err := c.retryPolicy.Do(ctx, true, func() (*http.Response, error) {
		httpReq, err := http.NewRequestWithContext(ctx, method, requestURL.String(), bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		httpReq.Host = signHost
		httpReq.Header = headers.Clone()
		return c.httpClient.Do(httpReq)
	})
	if err != nil {
		return err
	}
	defer httpclient.CloseResponse(httpResp)

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("read tos response: %w", err)
	}
	if err := decodeError(httpResp.StatusCode, httpResp.Header, respBody); err != nil {
		return err
	}
	if out == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode tos response: %w", err)
	}
	return nil
}

func (c *Client) signHeaders(method, host, path string, query url.Values, body []byte, extra http.Header) (http.Header, error) {
	timestamp := c.now().UTC()
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	shortDate := timestamp.Format("20060102")
	xDate := timestamp.Format(signDateFormat)
	payloadHash := hashSHA256Hex(body)

	headers := http.Header{}
	headers.Set(headerDate, timestamp.Format(http.TimeFormat))
	headers.Set(headerXDate, xDate)
	headers.Set(headerXContentSHA256, payloadHash)
	if token := strings.TrimSpace(c.credential.SessionToken); token != "" {
		headers.Set(headerSecurityToken, token)
	}

	canonicalHeaders := map[string]string{
		"host":                 host,
		"x-tos-content-sha256": payloadHash,
		"x-tos-date":           xDate,
	}
	if token := strings.TrimSpace(c.credential.SessionToken); token != "" {
		canonicalHeaders["x-tos-security-token"] = token
	}
	// Fold any caller-supplied x-tos-* / content-md5 / content-type headers
	// into the canonical set so the signature covers them.
	for key, values := range extra {
		lower := strings.ToLower(strings.TrimSpace(key))
		if !strings.HasPrefix(lower, "x-tos-") &&
			lower != "content-md5" && lower != "content-type" {
			continue
		}
		if len(values) == 0 {
			continue
		}
		canonicalHeaders[lower] = strings.TrimSpace(values[0])
	}

	signedHeaders := signedHeaderNames(canonicalHeaders)
	canonicalRequest := strings.Join([]string{
		method,
		canonicalURI(path),
		canonicalQueryString(query),
		canonicalHeadersText(canonicalHeaders, signedHeaders),
		strings.Join(signedHeaders, ";"),
		payloadHash,
	}, "\n")
	credentialScope := shortDate + "/" + normalizeRegionFromHost(host) + "/tos/request"
	stringToSign := strings.Join([]string{
		signAlgorithm,
		xDate,
		credentialScope,
		hashSHA256Hex([]byte(canonicalRequest)),
	}, "\n")
	signature := hex.EncodeToString(signTOS(c.credential.SecretKey, shortDate, normalizeRegionFromHost(host), stringToSign))

	headers.Set(
		headerAuthorization,
		fmt.Sprintf(
			"%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
			signAlgorithm,
			strings.TrimSpace(c.credential.AccessKey),
			credentialScope,
			strings.Join(signedHeaders, ";"),
			signature,
		),
	)
	return headers, nil
}

func (c *Client) requestURL(host, path string, query url.Values) (*url.URL, error) {
	base := &url.URL{
		Scheme: "https",
		Host:   host,
		Path:   httpclient.EnsureLeadingSlash(path),
	}
	if c.baseURL != nil {
		base = &url.URL{
			Scheme: c.baseURL.Scheme,
			Host:   c.baseURL.Host,
			Path:   httpclient.JoinPath(c.baseURL.Path, httpclient.EnsureLeadingSlash(path)),
		}
	}
	base.RawQuery = canonicalQueryString(query)
	return base, nil
}

func normalizeRegionFromHost(host string) string {
	host = normalizeHost(host)
	if strings.HasPrefix(host, "tos-") {
		if idx := strings.Index(host, "."); idx > len("tos-") {
			return strings.TrimPrefix(host[:idx], "tos-")
		}
	}
	if idx := strings.Index(host, ".tos-"); idx > 0 {
		rest := host[idx+1:]
		if end := strings.Index(rest, "."); end > len("tos-") {
			return strings.TrimPrefix(rest[:end], "tos-")
		}
	}
	return volcapi.DefaultRegion
}

func serviceHost(region string) string {
	return "tos-" + normalizeRegion(region) + ".volces.com"
}

func bucketHost(bucket, region string) string {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return serviceHost(region)
	}
	return bucket + "." + serviceHost(region)
}

func normalizeRegion(region string) string {
	region = strings.TrimSpace(region)
	if region == "" || region == "all" {
		return volcapi.DefaultRegion
	}
	return region
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
	return strings.TrimSuffix(strings.TrimSuffix(host, ":80"), ":443")
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
		case strings.HasPrefix(name, "x-tos-"):
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func canonicalHeadersText(headers map[string]string, signedHeaders []string) string {
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

func isUnreserved(c byte) bool {
	return ('A' <= c && c <= 'Z') ||
		('a' <= c && c <= 'z') ||
		('0' <= c && c <= '9') ||
		c == '-' || c == '_' || c == '.' || c == '~'
}

func hashSHA256Hex(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func signTOS(secretKey, shortDate, region, stringToSign string) []byte {
	dateKey := hmacSHA256([]byte(secretKey), shortDate)
	regionKey := hmacSHA256(dateKey, region)
	serviceKey := hmacSHA256(regionKey, "tos")
	signingKey := hmacSHA256(serviceKey, "request")
	return hmacSHA256(signingKey, stringToSign)
}

func hmacSHA256(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

var upperhex = "0123456789ABCDEF"
