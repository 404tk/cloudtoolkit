package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/auth"
)

type Request struct {
	Service    string
	Region     string
	Method     string
	Version    string
	Path       string
	Query      url.Values
	Body       []byte
	Headers    http.Header
	Idempotent bool
}

type Option func(*Client)

type Client struct {
	credential  auth.Credential
	httpClient  *http.Client
	retryPolicy RetryPolicy
	now         func() time.Time
	nonce       func() (string, error)
	baseURL     *url.URL
}

func NewClient(credential auth.Credential, opts ...Option) *Client {
	client := &Client{
		credential:  credential,
		httpClient:  NewHTTPClient(),
		retryPolicy: DefaultRetryPolicy(),
		now:         time.Now,
		nonce:       NewUUIDv4,
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

func WithNonceFunc(fn func() string) Option {
	return func(c *Client) {
		if fn == nil {
			return
		}
		c.nonce = func() (string, error) {
			return fn(), nil
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

func (c *Client) DoJSON(ctx context.Context, req Request, out any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.credential.Validate(); err != nil {
		return err
	}

	service := strings.ToLower(strings.TrimSpace(req.Service))
	if service == "" {
		return fmt.Errorf("jdcloud client: empty service")
	}
	version := strings.TrimSpace(req.Version)
	if version == "" {
		return fmt.Errorf("jdcloud client: empty version")
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}
	logicalHost := ResolveHost(service)
	if logicalHost == "" {
		return fmt.Errorf("jdcloud client: empty host")
	}
	signRegion := ResolveSigningRegion(req.Region)
	requestPath := joinPath("/", version, req.Path)
	query := cloneValues(req.Query)
	body := append([]byte(nil), req.Body...)
	headers := cloneHeader(req.Headers)
	contentType := strings.TrimSpace(headers.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/json"
	}
	headers.Set("Content-Type", contentType)
	if token := strings.TrimSpace(c.credential.SessionToken); token != "" {
		headers.Set("X-Jdcloud-Security-Token", token)
	}
	nonce, err := c.nonce()
	if err != nil {
		return err
	}

	signed, err := Sign(SignInput{
		Method:       method,
		Host:         logicalHost,
		Path:         requestPath,
		Query:        cloneValues(query),
		Body:         body,
		ContentType:  contentType,
		Service:      service,
		Region:       signRegion,
		AccessKey:    c.credential.AccessKey,
		SecretKey:    c.credential.SecretKey,
		SessionToken: c.credential.SessionToken,
		Nonce:        nonce,
		Timestamp:    c.now().UTC(),
		Headers:      headers,
	})
	if err != nil {
		return err
	}

	finalHeaders := headers.Clone()
	finalHeaders.Set(HeaderAuthorization, signed.Authorization)
	finalHeaders.Set(HeaderXJdcloudDate, signed.XJdcloudDate)
	finalHeaders.Set(HeaderXJdcloudNonce, signed.XJdcloudNonce)
	finalHeaders.Del("Host")

	scheme, networkHost, fullPath, err := c.resolveNetworkEndpoint(logicalHost, requestPath)
	if err != nil {
		return err
	}
	requestURL := url.URL{
		Scheme:   scheme,
		Host:     networkHost,
		Path:     fullPath,
		RawQuery: canonicalQuery(query),
	}

	idempotent := req.Idempotent || method == http.MethodGet || method == http.MethodHead || method == http.MethodPut || method == http.MethodDelete
	httpResp, err := c.retryPolicy.Do(ctx, idempotent, func() (*http.Response, error) {
		httpReq, err := http.NewRequestWithContext(ctx, method, requestURL.String(), bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		httpReq.Host = logicalHost
		httpReq.Header = finalHeaders.Clone()
		return c.httpClient.Do(httpReq)
	})
	if err != nil {
		return err
	}
	defer closeResponse(httpResp)

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("read jdcloud response: %w", err)
	}
	if err := annotateError(DecodeError(httpResp.StatusCode, respBody), service, ""); err != nil {
		return err
	}
	if out == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode jdcloud response: %w", err)
	}
	return nil
}

func (c *Client) resolveNetworkEndpoint(logicalHost, requestPath string) (scheme, host, fullPath string, err error) {
	if c.baseURL == nil {
		return "https", logicalHost, requestPath, nil
	}
	if strings.TrimSpace(c.baseURL.Scheme) == "" || strings.TrimSpace(c.baseURL.Host) == "" {
		return "", "", "", fmt.Errorf("jdcloud client: invalid base url %q", c.baseURL.String())
	}
	return c.baseURL.Scheme, c.baseURL.Host, joinPath(c.baseURL.Path, requestPath), nil
}

func cloneValues(values url.Values) url.Values {
	if len(values) == 0 {
		return url.Values{}
	}
	cloned := make(url.Values, len(values))
	for key, items := range values {
		cloned[key] = append([]string(nil), items...)
	}
	return cloned
}

func cloneHeader(headers http.Header) http.Header {
	if len(headers) == 0 {
		return http.Header{}
	}
	return headers.Clone()
}

func joinPath(parts ...string) string {
	result := ""
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if result == "" {
			result = ensureLeadingSlash(part)
			continue
		}
		result = strings.TrimRight(result, "/") + "/" + strings.TrimLeft(part, "/")
	}
	if result == "" {
		return "/"
	}
	return result
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
