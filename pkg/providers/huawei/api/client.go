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

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/endpoint"
)

type Request struct {
	Service    string
	Region     string
	Intl       bool
	Method     string
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
	baseURL     *url.URL
}

func NewClient(cred auth.Credential, opts ...Option) *Client {
	client := &Client{
		credential:  cred,
		httpClient:  NewHTTPClient(),
		retryPolicy: DefaultRetryPolicy(),
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

func WithRetryPolicy(p RetryPolicy) Option {
	return func(c *Client) {
		if p != nil {
			c.retryPolicy = p
		}
	}
}

func WithClock(now func() time.Time) Option {
	return func(c *Client) {
		if now != nil {
			c.now = now
		}
	}
}

func WithBaseURL(raw string) Option {
	return func(c *Client) {
		if raw == "" {
			return
		}
		if u, err := url.Parse(raw); err == nil {
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

	service := strings.TrimSpace(strings.ToLower(req.Service))
	if service == "" {
		return fmt.Errorf("huawei client: empty service")
	}
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}
	scheme, host, path, err := c.resolveEndpoint(req, service)
	if err != nil {
		return err
	}

	body := append([]byte(nil), req.Body...)
	headers := cloneHeader(req.Headers)
	if len(body) > 0 && strings.TrimSpace(headers.Get("Content-Type")) == "" {
		headers.Set("Content-Type", "application/json;charset=UTF-8")
	}
	timestamp := c.now().UTC()
	signed, err := Sign(&SignRequest{
		Method:    method,
		Host:      host,
		Path:      path,
		Query:     cloneValues(req.Query),
		Headers:   flattenHeaders(headers),
		Body:      body,
		AccessKey: c.credential.AK,
		SecretKey: c.credential.SK,
		Timestamp: timestamp,
	})
	if err != nil {
		return err
	}

	finalHeaders := headers.Clone()
	for key, value := range signed {
		if value == "" {
			continue
		}
		finalHeaders.Set(key, value)
	}
	removeReservedHeader(finalHeaders, "Host")

	requestURL := url.URL{
		Scheme:   scheme,
		Host:     host,
		Path:     path,
		RawQuery: cloneValues(req.Query).Encode(),
	}

	idempotent := req.Idempotent || method == http.MethodGet
	httpResp, err := c.retryPolicy.Do(ctx, idempotent, func() (*http.Response, error) {
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
	defer closeResponse(httpResp)

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("read huawei response: %w", err)
	}
	if err := withRequestID(DecodeError(httpResp.StatusCode, respBody), httpResp.Header.Get("X-Request-Id")); err != nil {
		return err
	}
	if out == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode huawei response: %w", err)
	}
	return nil
}

func (c *Client) resolveEndpoint(req Request, service string) (string, string, string, error) {
	base := endpoint.For(service, strings.TrimSpace(req.Region), req.Intl)
	if c.baseURL != nil {
		base = c.baseURL.String()
	}
	if strings.TrimSpace(base) == "" {
		return "", "", "", fmt.Errorf("huawei client: empty endpoint")
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", "", "", fmt.Errorf("huawei client: invalid endpoint %q: %w", base, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", "", "", fmt.Errorf("huawei client: invalid endpoint %q", base)
	}
	return u.Scheme, u.Host, joinPath(u.Path, ensureLeadingSlash(req.Path)), nil
}

func flattenHeaders(headers http.Header) map[string]string {
	flattened := make(map[string]string, len(headers))
	for key, values := range headers {
		name := strings.TrimSpace(key)
		if name == "" || isReservedHeader(name) {
			continue
		}
		flattened[name] = strings.Join(values, ",")
	}
	return flattened
}

func cloneHeader(headers http.Header) http.Header {
	if headers == nil {
		return http.Header{}
	}
	return headers.Clone()
}

func cloneValues(values url.Values) url.Values {
	if values == nil {
		return url.Values{}
	}
	cloned := make(url.Values, len(values))
	for key, items := range values {
		cloned[key] = append([]string(nil), items...)
	}
	return cloned
}

func joinPath(basePath, requestPath string) string {
	switch {
	case basePath == "", basePath == "/":
		return ensureLeadingSlash(requestPath)
	case requestPath == "", requestPath == "/":
		return ensureLeadingSlash(basePath)
	default:
		return strings.TrimRight(basePath, "/") + ensureLeadingSlash(requestPath)
	}
}

func isReservedHeader(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case strings.ToLower(HeaderAuthorization), strings.ToLower(HeaderXDate), strings.ToLower(HeaderContentSha256), "host":
		return true
	default:
		return false
	}
}

func removeReservedHeader(headers http.Header, name string) {
	for key := range headers {
		if strings.EqualFold(key, name) {
			headers.Del(key)
		}
	}
}
