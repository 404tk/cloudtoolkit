package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/azure/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/cloud"
	"github.com/404tk/cloudtoolkit/pkg/providers/internal/httpclient"
)

type Request struct {
	Method     string
	Path       string
	Query      url.Values
	Headers    http.Header
	Body       []byte
	Idempotent bool
}

type Option func(*Client)

type Client struct {
	tokenSource *auth.TokenSource
	endpoints   cloud.Endpoints
	httpClient  *http.Client
	retryPolicy RetryPolicy
	baseURL     *url.URL
}

func NewClient(ts *auth.TokenSource, endpoints cloud.Endpoints, opts ...Option) *Client {
	client := &Client{
		tokenSource: ts,
		endpoints:   endpoints,
		httpClient:  NewHTTPClient(),
		retryPolicy: DefaultRetryPolicy(),
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

func WithBaseURL(raw string) Option {
	return func(c *Client) {
		if strings.TrimSpace(raw) == "" {
			return
		}
		if u, err := url.Parse(raw); err == nil {
			c.baseURL = u
		}
	}
}

func (c *Client) Do(ctx context.Context, req Request, out any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if c.tokenSource == nil {
		return fmt.Errorf("azure client: nil token source")
	}
	token, err := c.tokenSource.Token(ctx)
	if err != nil {
		return err
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}
	requestURL, err := c.resolveURL(req.Path, req.Query)
	if err != nil {
		return err
	}

	headers := httpclient.CloneHeader(req.Headers)
	headers.Set("Authorization", "Bearer "+token.AccessToken)
	if len(req.Body) > 0 && strings.TrimSpace(headers.Get("Content-Type")) == "" {
		headers.Set("Content-Type", "application/json")
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, requestURL.String(), bytes.NewReader(req.Body))
	if err != nil {
		return err
	}
	httpReq.Header = headers.Clone()

	httpResp, body, err := httpclient.Execute(ctx, c.httpClient, c.retryPolicy, httpReq, req.Idempotent)
	if err != nil {
		return err
	}
	if err := withRequestID(DecodeError(httpResp.StatusCode, body), httpResp.Header.Get("x-ms-request-id")); err != nil {
		return err
	}
	return httpclient.DecodeJSON(httpResp, body, "azure", out)
}

func (c *Client) resolveURL(path string, query url.Values) (*url.URL, error) {
	base := strings.TrimSpace(c.endpoints.ResourceManager)
	if c.baseURL != nil {
		base = c.baseURL.String()
	}
	if base == "" {
		return nil, fmt.Errorf("azure client: empty resource manager endpoint")
	}

	if parsed, err := url.Parse(strings.TrimSpace(path)); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		parsed.RawQuery = httpclient.CloneValues(query).Encode()
		return parsed, nil
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("azure client: invalid base url %q: %w", base, err)
	}
	baseURL.Path = httpclient.JoinPath(baseURL.Path, path)
	baseURL.RawQuery = httpclient.CloneValues(query).Encode()
	return baseURL, nil
}
