package api

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/internal/httpclient"
)

type Request struct {
	Service    string
	Region     string
	Action     string
	Version    string
	Method     string
	Path       string
	Query      url.Values
	Body       []byte
	Headers    http.Header
	Idempotent bool
	Scheme     string
	Host       string
}

type Option func(*Client)

type Client struct {
	credential  auth.Credential
	httpClient  *http.Client
	retryPolicy RetryPolicy
	signer      SigV4Signer
	now         func() time.Time
	baseURL     *url.URL
}

func NewClient(credential auth.Credential, opts ...Option) *Client {
	client := &Client{
		credential:  credential,
		httpClient:  NewHTTPClient(),
		retryPolicy: DefaultRetryPolicy(),
		now:         time.Now,
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if client != nil {
			c.httpClient = client
		}
	}
}

func WithRetryPolicy(policy RetryPolicy) Option {
	return func(c *Client) {
		c.retryPolicy = policy
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

func (c *Client) DoXML(ctx context.Context, req Request, resp any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.credential.Validate(); err != nil {
		return err
	}
	service := strings.TrimSpace(strings.ToLower(req.Service))
	if service == "" {
		return fmt.Errorf("aws client: empty service")
	}
	version := strings.TrimSpace(req.Version)
	if version == "" {
		return fmt.Errorf("aws client: empty version")
	}
	action := strings.TrimSpace(req.Action)
	if action == "" {
		return fmt.Errorf("aws client: empty action")
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodPost
	}
	region := normalizeServiceRegion(service, req.Region)
	bodyValues := httpclient.CloneValues(req.Query)
	bodyValues.Set("Action", action)
	bodyValues.Set("Version", version)
	body := []byte(bodyValues.Encode())

	scheme, host, path := c.resolveEndpoint(req, service, region)
	timestamp := c.now().UTC()
	contentType := "application/x-www-form-urlencoded; charset=utf-8"
	signed, err := c.signer.Sign(c.credential, SignInput{
		Method:      method,
		Service:     service,
		Region:      region,
		Host:        host,
		Path:        path,
		ContentType: contentType,
		Payload:     body,
		Timestamp:   timestamp,
		Headers:     req.Headers,
	})
	if err != nil {
		return err
	}

	headers := c.buildHeaders(req.Headers, contentType, signed)
	requestURL := url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   path,
	}

	httpResp, err := c.retryPolicy.Do(ctx, req.Idempotent, func() (*http.Response, error) {
		httpReq, err := http.NewRequestWithContext(ctx, method, requestURL.String(), bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		httpReq.Host = host
		httpReq.Header = headers.Clone()
		return c.httpClient.Do(httpReq)
	})
	if err != nil {
		return err
	}
	defer httpclient.CloseResponse(httpResp)

	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("read aws response: %w", err)
	}
	if err := DecodeError(httpResp.StatusCode, responseBody); err != nil {
		return err
	}
	if resp == nil || len(responseBody) == 0 {
		return nil
	}
	if err := xml.Unmarshal(responseBody, resp); err != nil {
		return fmt.Errorf("decode aws response: %w", err)
	}
	return nil
}

func (c *Client) DoRESTXML(ctx context.Context, req Request, resp any) error {
	return c.doREST(ctx, req, resp, decodeXMLBody)
}

// DoRESTJSON sends req as a JSON-bodied REST call (used by SSM, ECS-style
// services that speak JSON-1.1). Caller provides Headers including
// `Content-Type: application/x-amz-json-1.1` and `X-Amz-Target: <service>.<action>`.
func (c *Client) DoRESTJSON(ctx context.Context, req Request, resp any) error {
	return c.doREST(ctx, req, resp, decodeJSONBody)
}

func decodeXMLBody(body []byte, out any) error {
	if err := xml.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode aws response: %w", err)
	}
	return nil
}

func decodeJSONBody(body []byte, out any) error {
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode aws response: %w", err)
	}
	return nil
}

func (c *Client) doREST(ctx context.Context, req Request, resp any, decode func([]byte, any) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.credential.Validate(); err != nil {
		return err
	}
	service := strings.TrimSpace(strings.ToLower(req.Service))
	if service == "" {
		return fmt.Errorf("aws client: empty service")
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}
	region := normalizeServiceRegion(service, req.Region)
	body := append([]byte(nil), req.Body...)
	query := httpclient.CloneValues(req.Query)
	headers := httpclient.CloneHeader(req.Headers)
	if service == "s3" && strings.TrimSpace(headers.Get("X-Amz-Content-Sha256")) == "" {
		headers.Set("X-Amz-Content-Sha256", hashSHA256Hex(body))
	}
	scheme, host, path := c.resolveEndpoint(req, service, region)
	timestamp := c.now().UTC()
	contentType := strings.TrimSpace(headers.Get("Content-Type"))
	signed, err := c.signer.Sign(c.credential, SignInput{
		Method:      method,
		Service:     service,
		Region:      region,
		Host:        host,
		Path:        path,
		Query:       query,
		ContentType: contentType,
		Payload:     body,
		Timestamp:   timestamp,
		Headers:     headers,
	})
	if err != nil {
		return err
	}

	finalHeaders := c.buildHeaders(headers, contentType, signed)
	requestURL := url.URL{
		Scheme:   scheme,
		Host:     host,
		Path:     path,
		RawQuery: query.Encode(),
	}

	httpResp, err := c.retryPolicy.Do(ctx, req.Idempotent, func() (*http.Response, error) {
		httpReq, err := http.NewRequestWithContext(ctx, method, requestURL.String(), bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		httpReq.Host = host
		httpReq.Header = finalHeaders.Clone()
		return c.httpClient.Do(httpReq)
	})
	if err != nil {
		return err
	}
	defer httpclient.CloseResponse(httpResp)

	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("read aws response: %w", err)
	}
	if err := DecodeError(httpResp.StatusCode, responseBody); err != nil {
		return err
	}
	if resp == nil || len(responseBody) == 0 {
		return nil
	}
	return decode(responseBody, resp)
}

func (c *Client) resolveEndpoint(req Request, service, region string) (string, string, string) {
	scheme := "https"
	host := defaultHost(service, region)
	path := httpclient.EnsureLeadingSlash(req.Path)

	if c.baseURL != nil {
		if c.baseURL.Scheme != "" {
			scheme = c.baseURL.Scheme
		}
		if c.baseURL.Host != "" {
			host = c.baseURL.Host
		}
		path = httpclient.JoinPath(c.baseURL.Path, path)
	}
	if req.Scheme != "" {
		scheme = req.Scheme
	}
	if req.Host != "" {
		host = req.Host
	}
	return scheme, host, path
}

func (c *Client) buildHeaders(extra http.Header, contentType string, signed Signature) http.Header {
	headers := http.Header{}
	if strings.TrimSpace(contentType) != "" {
		headers.Set("Content-Type", contentType)
	}
	headers.Set("X-Amz-Date", signed.AmzDate)
	headers.Set("Authorization", signed.Authorization)
	if token := strings.TrimSpace(c.credential.SessionToken); token != "" {
		headers.Set("X-Amz-Security-Token", token)
	}
	for key, values := range extra {
		canonicalKey := http.CanonicalHeaderKey(key)
		if isReservedHeader(canonicalKey) {
			continue
		}
		for _, value := range values {
			headers.Add(canonicalKey, value)
		}
	}
	return headers
}

func normalizeRegion(region string) string {
	region = strings.TrimSpace(region)
	if region == "" || region == "all" {
		return "us-east-1"
	}
	return region
}

func normalizeServiceRegion(service, region string) string {
	region = normalizeRegion(region)
	switch service {
	case "iam":
		return normalizeIAMRegion(region)
	case "sts":
		return normalizeSTSRegion(region)
	default:
		return region
	}
}

func defaultHost(service, region string) string {
	if service == "sts" {
		if strings.HasPrefix(region, "cn-") {
			return service + "." + region + ".amazonaws.com.cn"
		}
		return service + "." + region + ".amazonaws.com"
	}
	if service == "iam" {
		if strings.HasPrefix(region, "cn-") {
			return "iam.cn-north-1.amazonaws.com.cn"
		}
		return "iam.amazonaws.com"
	}
	suffix := "amazonaws.com"
	if strings.HasPrefix(region, "cn-") {
		suffix = "amazonaws.com.cn"
	}
	return service + "." + region + "." + suffix
}

func isReservedHeader(key string) bool {
	switch http.CanonicalHeaderKey(key) {
	case "Authorization", "Content-Type", "X-Amz-Date", "X-Amz-Security-Token":
		return true
	default:
		return false
	}
}
