package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/internal/httpclient"
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
		c.retryPolicy = p
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

	headers := httpclient.CloneHeader(req.Headers)
	headers.Set("Authorization", "Bearer "+token.AccessToken)
	if req.Body != nil && strings.TrimSpace(headers.Get("Content-Type")) == "" {
		headers.Set("Content-Type", "application/json; charset=UTF-8")
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewReader(req.Body))
	if err != nil {
		return err
	}
	httpReq.Header = headers.Clone()

	httpResp, body, err := httpclient.Execute(ctx, c.httpClient, c.retryPolicy, httpReq, req.Idempotent)
	if err != nil {
		return err
	}
	if err := DecodeError(httpResp.StatusCode, body); err != nil {
		return err
	}
	return httpclient.DecodeJSON(httpResp, body, "gcp", out)
}

func resolveURL(baseURL, path string, query url.Values) (string, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return "", fmt.Errorf("gcp client: empty base url")
	}
	parsed, err := url.Parse(baseURL + httpclient.EnsureLeadingSlash(path))
	if err != nil {
		return "", err
	}
	parsed.RawQuery = httpclient.CloneValues(query).Encode()
	return parsed.String(), nil
}
