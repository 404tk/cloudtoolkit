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

	"github.com/404tk/cloudtoolkit/pkg/providers/azure/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/azure/cloud"
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
		if p != nil {
			c.retryPolicy = p
		}
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

	headers := cloneHeader(req.Headers)
	headers.Set("Authorization", "Bearer "+token.AccessToken)
	if len(req.Body) > 0 && strings.TrimSpace(headers.Get("Content-Type")) == "" {
		headers.Set("Content-Type", "application/json")
	}

	httpResp, err := c.retryPolicy.Do(ctx, req.Idempotent, func() (*http.Response, error) {
		httpReq, err := http.NewRequestWithContext(ctx, method, requestURL.String(), bytes.NewReader(req.Body))
		if err != nil {
			return nil, err
		}
		httpReq.Header = headers.Clone()
		return c.httpClient.Do(httpReq)
	})
	if err != nil {
		return err
	}
	defer closeResponse(httpResp)

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("read azure response: %w", err)
	}
	if err := withRequestID(DecodeError(httpResp.StatusCode, body), httpResp.Header.Get("x-ms-request-id")); err != nil {
		return err
	}
	if out == nil || len(body) == 0 || httpResp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode azure response: %w", err)
	}
	return nil
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
		parsed.RawQuery = cloneValues(query).Encode()
		return parsed, nil
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("azure client: invalid base url %q: %w", base, err)
	}
	baseURL.Path = joinPath(baseURL.Path, ensureLeadingSlash(path))
	baseURL.RawQuery = cloneValues(query).Encode()
	return baseURL, nil
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

func closeResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 8<<10))
	_ = resp.Body.Close()
}
