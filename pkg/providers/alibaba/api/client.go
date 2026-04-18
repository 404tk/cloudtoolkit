package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
)

type Request struct {
	Product    string
	Version    string
	Action     string
	Region     string
	Method     string
	Query      url.Values
	Headers    http.Header
	Idempotent bool
	Scheme     string
	Host       string
	Path       string
}

type Option func(*Client)

type Client struct {
	credential  auth.Credential
	httpClient  *http.Client
	retryPolicy RetryPolicy
	signer      RPCSigner
	now         func() time.Time
	nonce       func() string
	baseURL     *url.URL
	userAgent   string
}

func NewClient(credential auth.Credential, opts ...Option) *Client {
	client := &Client{
		credential:  credential,
		httpClient:  NewHTTPClient(),
		retryPolicy: DefaultRetryPolicy(),
		now:         time.Now,
		nonce:       defaultNonce,
		userAgent:   "ctk",
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

func WithNonce(fn func() string) Option {
	return func(c *Client) {
		if fn != nil {
			c.nonce = fn
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

func WithUserAgent(userAgent string) Option {
	return func(c *Client) {
		c.userAgent = strings.TrimSpace(userAgent)
	}
}

func (c *Client) Do(ctx context.Context, req Request, resp any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.credential.Validate(); err != nil {
		return err
	}
	product := strings.TrimSpace(req.Product)
	if product == "" {
		return fmt.Errorf("alibaba client: empty product")
	}
	version := strings.TrimSpace(req.Version)
	if version == "" {
		return fmt.Errorf("alibaba client: empty version")
	}
	action := strings.TrimSpace(req.Action)
	if action == "" {
		return fmt.Errorf("alibaba client: empty action")
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodPost
	}
	region := NormalizeRegion(req.Region)
	params := cloneValues(req.Query)
	params.Set("Version", version)
	params.Set("Action", action)
	if _, ok := params["Format"]; !ok {
		params.Set("Format", "JSON")
	}
	params.Set("Timestamp", c.now().UTC().Format("2006-01-02T15:04:05Z"))
	params.Set("SignatureMethod", "HMAC-SHA1")
	params.Set("SignatureVersion", "1.0")
	params.Set("SignatureType", "")
	params.Set("SignatureNonce", c.nonce())
	params.Set("AccessKeyId", c.credential.AccessKeyID)
	if _, ok := params["RegionId"]; !ok {
		params.Set("RegionId", region)
	}
	if c.credential.SecurityToken != "" {
		params.Set("SecurityToken", c.credential.SecurityToken)
	}

	signedParams, err := c.signer.Sign(c.credential, SignInput{
		Method: method,
		Params: params,
	})
	if err != nil {
		return err
	}

	scheme, host, path, err := c.resolveEndpoint(req, region)
	if err != nil {
		return err
	}
	requestURL := url.URL{
		Scheme:   scheme,
		Host:     host,
		Path:     path,
		RawQuery: signedParams.Encode(),
	}
	headers := c.buildHeaders(req.Headers)

	httpResp, err := c.retryPolicy.Do(ctx, req.Idempotent, func() (*http.Response, error) {
		httpReq, err := http.NewRequestWithContext(ctx, method, requestURL.String(), nil)
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
	defer closeResponse(httpResp)

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("read alibaba response: %w", err)
	}
	if err := DecodeError(httpResp.StatusCode, body); err != nil {
		return err
	}
	if resp == nil || len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, resp); err != nil {
		return fmt.Errorf("decode alibaba response: %w", err)
	}
	return nil
}

func (c *Client) resolveEndpoint(req Request, region string) (string, string, string, error) {
	scheme := "https"
	path := "/"
	host := ""
	if c.baseURL != nil {
		if c.baseURL.Scheme != "" {
			scheme = c.baseURL.Scheme
		}
		if c.baseURL.Host != "" {
			host = c.baseURL.Host
		}
		path = joinPath(c.baseURL.Path, path)
	}
	if req.Path != "" {
		path = joinPath(path, req.Path)
	}
	if req.Scheme != "" {
		scheme = req.Scheme
	}
	if req.Host != "" {
		host = req.Host
	}
	if host == "" {
		var err error
		host, err = resolveEndpointHost(req.Product, region)
		if err != nil {
			return "", "", "", err
		}
	}
	return scheme, host, path, nil
}

func (c *Client) buildHeaders(extra http.Header) http.Header {
	headers := http.Header{}
	headers.Set("Accept", "application/json")
	headers.Set("Content-Type", "application/x-www-form-urlencoded")
	if c.userAgent != "" {
		headers.Set("User-Agent", c.userAgent)
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

func cloneValues(values url.Values) url.Values {
	cloned := url.Values{}
	for key, items := range values {
		copied := make([]string, len(items))
		copy(copied, items)
		cloned[key] = copied
	}
	return cloned
}

func ensureLeadingSlash(path string) string {
	if path == "" {
		return "/"
	}
	if strings.HasPrefix(path, "/") {
		return path
	}
	return "/" + path
}

func joinPath(basePath, requestPath string) string {
	basePath = strings.TrimRight(basePath, "/")
	requestPath = ensureLeadingSlash(requestPath)
	switch {
	case basePath == "":
		return requestPath
	case requestPath == "/":
		return basePath
	default:
		return basePath + requestPath
	}
}

func isReservedHeader(key string) bool {
	switch strings.ToLower(key) {
	case "host", "accept", "content-type", "user-agent":
		return true
	default:
		return false
	}
}
