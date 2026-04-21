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

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/auth"
)

type Request struct {
	Method     string
	BaseURL    string
	Path       string
	Query      url.Values
	Headers    http.Header
	Body       []byte
	Idempotent bool
}

type Option func(*Client)

type Client struct {
	tokenSource *auth.TokenSource
	httpClient  *http.Client
	retryPolicy RetryPolicy
}

func NewClient(ts *auth.TokenSource, opts ...Option) *Client {
	client := &Client{
		tokenSource: ts,
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

func (c *Client) Do(ctx context.Context, req Request, out any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if c.tokenSource == nil {
		return fmt.Errorf("gcp client: nil token source")
	}

	token, err := c.tokenSource.Token(ctx)
	if err != nil {
		return err
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}
	requestURL, err := resolveURL(req.BaseURL, req.Path, req.Query)
	if err != nil {
		return err
	}

	headers := cloneHeader(req.Headers)
	headers.Set("Authorization", "Bearer "+token.AccessToken)
	if req.Body != nil && strings.TrimSpace(headers.Get("Content-Type")) == "" {
		headers.Set("Content-Type", "application/json; charset=UTF-8")
	}

	httpResp, err := c.retryPolicy.Do(ctx, req.Idempotent, func() (*http.Response, error) {
		httpReq, err := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewReader(req.Body))
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
		return fmt.Errorf("read gcp response: %w", err)
	}
	if err := DecodeError(httpResp.StatusCode, body); err != nil {
		return err
	}
	if out == nil || len(body) == 0 || httpResp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode gcp response: %w", err)
	}
	return nil
}

func resolveURL(baseURL, path string, query url.Values) (string, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return "", fmt.Errorf("gcp client: empty base url")
	}
	parsed, err := url.Parse(baseURL + ensureLeadingSlash(path))
	if err != nil {
		return "", err
	}
	parsed.RawQuery = cloneValues(query).Encode()
	return parsed.String(), nil
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

func closeResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 8<<10))
	_ = resp.Body.Close()
}
