package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/internal/httpclient"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
)

type Request struct {
	Service         string
	Version         string
	Action          string
	Region          string
	Method          string
	Path            string
	Query           url.Values
	Headers         http.Header
	Body            any
	Idempotent      bool
	UnsignedPayload bool
	Scheme          string
	Host            string
}

type Option func(*Client)

type Client struct {
	credential    auth.Credential
	httpClient    *http.Client
	retryPolicy   RetryPolicy
	signer        TC3Signer
	now           func() time.Time
	requestClient string
	language      string
	baseURL       *url.URL
}

func NewClient(credential auth.Credential, opts ...Option) *Client {
	client := &Client{
		credential:    credential,
		httpClient:    NewHTTPClient(),
		retryPolicy:   DefaultRetryPolicy(),
		now:           time.Now,
		requestClient: "ctk",
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

func WithRequestClient(name string) Option {
	return func(c *Client) {
		c.requestClient = strings.TrimSpace(name)
	}
}

func WithLanguage(language string) Option {
	return func(c *Client) {
		c.language = strings.TrimSpace(language)
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

func (c *Client) DoJSON(
	ctx context.Context,
	service string,
	version string,
	action string,
	region string,
	req any,
	resp any,
) error {
	return c.Do(ctx, Request{
		Service:    service,
		Version:    version,
		Action:     action,
		Region:     region,
		Method:     http.MethodPost,
		Path:       "/",
		Body:       req,
		Idempotent: true,
	}, resp)
}

func (c *Client) Do(ctx context.Context, req Request, resp any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.credential.Validate(); err != nil {
		return err
	}
	if req.Service == "" {
		return fmt.Errorf("tencent client: empty service")
	}
	if req.Version == "" {
		return fmt.Errorf("tencent client: empty version")
	}
	if req.Action == "" {
		return fmt.Errorf("tencent client: empty action")
	}

	method := req.Method
	if method == "" {
		method = http.MethodPost
	}
	method = strings.ToUpper(method)
	path := req.Path
	if path == "" {
		path = "/"
	}
	query := req.Query.Encode()
	payload, err := marshalBody(method, req.Body)
	if err != nil {
		return err
	}

	scheme, host, finalPath := c.resolveEndpoint(req.Host, req.Scheme, path, req.Service)
	contentType := "application/json"
	timestamp := c.now().UTC()
	signed, err := c.signer.Sign(c.credential, SignInput{
		Method:          method,
		Service:         req.Service,
		Host:            host,
		Path:            finalPath,
		Query:           query,
		ContentType:     contentType,
		Timestamp:       timestamp,
		Payload:         payload,
		UnsignedPayload: req.UnsignedPayload,
	})
	if err != nil {
		return err
	}

	headers := c.buildHeaders(req, contentType, timestamp, signed)
	requestURL := url.URL{
		Scheme:   scheme,
		Host:     host,
		Path:     finalPath,
		RawQuery: query,
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, requestURL.String(), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	httpReq.Host = host
	httpReq.Header = headers.Clone()

	httpResp, body, err := httpclient.Execute(ctx, c.httpClient, c.retryPolicy, httpReq, req.Idempotent)
	if err != nil {
		return err
	}
	if err := DecodeError(httpResp.StatusCode, body); err != nil {
		return err
	}
	return httpclient.DecodeJSON(httpResp, body, "tencent", resp)
}

func (c *Client) resolveEndpoint(overrideHost, overrideScheme, path, service string) (string, string, string) {
	scheme := "https"
	host := service + ".tencentcloudapi.com"
	finalPath := httpclient.EnsureLeadingSlash(path)

	if c.baseURL != nil {
		if c.baseURL.Scheme != "" {
			scheme = c.baseURL.Scheme
		}
		if c.baseURL.Host != "" {
			host = c.baseURL.Host
		}
		finalPath = httpclient.JoinPath(c.baseURL.Path, finalPath)
	}
	if overrideScheme != "" {
		scheme = overrideScheme
	}
	if overrideHost != "" {
		host = overrideHost
	}
	return scheme, host, finalPath
}

func (c *Client) buildHeaders(req Request, contentType string, timestamp time.Time, signed Signature) http.Header {
	headers := http.Header{}
	headers.Set("Content-Type", contentType)
	headers.Set("X-TC-Action", req.Action)
	headers.Set("X-TC-Version", req.Version)
	headers.Set("X-TC-Timestamp", strconv.FormatInt(timestamp.Unix(), 10))
	headers.Set("Authorization", signed.Authorization)
	if req.Region != "" {
		headers.Set("X-TC-Region", req.Region)
	}
	if c.requestClient != "" {
		headers.Set("X-TC-RequestClient", c.requestClient)
	}
	if c.language != "" {
		headers.Set("X-TC-Language", c.language)
	}
	if c.credential.Token != "" {
		headers.Set("X-TC-Token", c.credential.Token)
	}
	if req.UnsignedPayload {
		headers.Set("X-TC-Content-SHA256", "UNSIGNED-PAYLOAD")
	}
	for key, values := range req.Headers {
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

func marshalBody(method string, body any) ([]byte, error) {
	if method == http.MethodGet {
		return nil, nil
	}
	if body == nil {
		return []byte("{}"), nil
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode tencent request: %w", err)
	}
	return payload, nil
}

func isReservedHeader(key string) bool {
	switch strings.ToLower(key) {
	case "host",
		"content-type",
		"authorization",
		"x-tc-action",
		"x-tc-version",
		"x-tc-timestamp",
		"x-tc-requestclient",
		"x-tc-language",
		"x-tc-region",
		"x-tc-token",
		"x-tc-content-sha256":
		return true
	default:
		return false
	}
}
