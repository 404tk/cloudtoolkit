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

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/auth"
)

type Request struct {
	Service    string
	Version    string
	Action     string
	Method     string
	Region     string
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
	siteStack   string
	baseURL     *url.URL
}

func NewClient(cred auth.Credential, opts ...Option) *Client {
	client := &Client{
		credential:  cred,
		httpClient:  NewHTTPClient(),
		retryPolicy: DefaultRetryPolicy(),
		now:         time.Now,
		siteStack:   defaultSiteStack,
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

func WithSiteStack(siteStack string) Option {
	return func(c *Client) {
		if strings.TrimSpace(siteStack) != "" {
			c.siteStack = strings.TrimSpace(siteStack)
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

func (c *Client) DoOpenAPI(ctx context.Context, req Request, out any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.credential.Validate(); err != nil {
		return err
	}

	service := strings.ToLower(strings.TrimSpace(req.Service))
	if service == "" {
		return fmt.Errorf("volcengine client: empty service")
	}
	version := strings.TrimSpace(req.Version)
	if version == "" {
		return fmt.Errorf("volcengine client: empty version")
	}
	action := strings.TrimSpace(req.Action)
	if action == "" {
		return fmt.Errorf("volcengine client: empty action")
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}
	query := cloneValues(req.Query)
	query.Set("Action", action)
	query.Set("Version", version)

	signRegion := effectiveRegion(req.Region)
	endpointRegion := strings.TrimSpace(req.Region)
	if endpointRegion == "" || endpointRegion == "all" {
		endpointRegion = signRegion
	}
	scheme, host, path, err := c.resolveEndpoint(req, service, endpointRegion)
	if err != nil {
		return err
	}

	body := []byte(nil)
	if method == http.MethodPost {
		body = append([]byte(nil), req.Body...)
	}
	headers := cloneHeader(req.Headers)
	contentType := strings.TrimSpace(headers.Get("Content-Type"))
	if contentType == "" {
		switch {
		case method == http.MethodPost && len(body) > 0:
			contentType = "application/json; charset=utf-8"
		default:
			contentType = "application/x-www-form-urlencoded; charset=utf-8"
		}
	}
	headers.Set("Content-Type", contentType)
	if method == http.MethodPost && len(body) > 0 && strings.TrimSpace(headers.Get("Accept")) == "" {
		headers.Set("Accept", "application/json")
	}

	signed, err := Sign(SignInput{
		Method:       method,
		Host:         host,
		Path:         path,
		Query:        cloneValues(query),
		Body:         body,
		ContentType:  contentType,
		Service:      service,
		Region:       signRegion,
		AccessKey:    c.credential.AccessKey,
		SecretKey:    c.credential.SecretKey,
		SessionToken: c.credential.SessionToken,
		Headers:      headers,
		Timestamp:    c.now().UTC(),
	})
	if err != nil {
		return err
	}

	finalHeaders := headers.Clone()
	finalHeaders.Set(HeaderAuthorization, signed.Authorization)
	finalHeaders.Set(HeaderXDate, signed.XDate)
	finalHeaders.Set(HeaderXContentSHA256, signed.XContentSHA256)
	if token := strings.TrimSpace(c.credential.SessionToken); token != "" {
		finalHeaders.Set(HeaderXSecurityToken, token)
	}
	finalHeaders.Del("Host")

	requestURL := url.URL{
		Scheme:   scheme,
		Host:     host,
		Path:     path,
		RawQuery: canonicalQueryString(query),
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
		return fmt.Errorf("read volcengine response: %w", err)
	}
	if err := annotateError(DecodeError(httpResp.StatusCode, respBody), service, action); err != nil {
		return err
	}
	if out == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode volcengine response: %w", err)
	}
	return nil
}

func (c *Client) resolveEndpoint(req Request, service, region string) (string, string, string, error) {
	base := ResolveEndpoint(service, region, c.siteStack)
	if c.baseURL != nil {
		base = c.baseURL.String()
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", "", "", fmt.Errorf("volcengine client: invalid endpoint %q: %w", base, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", "", "", fmt.Errorf("volcengine client: invalid endpoint %q", base)
	}
	return u.Scheme, u.Host, joinPath(u.Path, ensureLeadingSlash(req.Path)), nil
}

func effectiveRegion(region string) string {
	region = strings.TrimSpace(region)
	if region == "" || region == "all" {
		return DefaultRegion
	}
	return region
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

func joinPath(base, path string) string {
	switch {
	case base == "" || base == "/":
		return ensureLeadingSlash(path)
	case path == "" || path == "/":
		return ensureLeadingSlash(base)
	default:
		return strings.TrimRight(base, "/") + ensureLeadingSlash(path)
	}
}
