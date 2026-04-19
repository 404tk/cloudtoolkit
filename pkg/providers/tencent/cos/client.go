package cos

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
)

const defaultServiceEndpoint = "http://service.cos.myqcloud.com"

type Option func(*Client)

type Client struct {
	credential      auth.Credential
	httpClient      *http.Client
	retryPolicy     api.RetryPolicy
	now             func() time.Time
	serviceEndpoint string
}

func NewClient(cred auth.Credential, opts ...Option) *Client {
	client := &Client{
		credential:      cred,
		httpClient:      api.NewHTTPClient(),
		retryPolicy:     api.DefaultRetryPolicy(),
		now:             time.Now,
		serviceEndpoint: defaultServiceEndpoint,
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

func WithRetryPolicy(p api.RetryPolicy) Option {
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

func WithServiceEndpoint(rawURL string) Option {
	return func(c *Client) {
		if strings.TrimSpace(rawURL) != "" {
			c.serviceEndpoint = rawURL
		}
	}
}

func (c *Client) ListBuckets(ctx context.Context) (*ListBucketsResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.credential.Validate(); err != nil {
		return nil, err
	}

	u, err := c.serviceURL()
	if err != nil {
		return nil, err
	}

	httpResp, err := c.retryPolicy.Do(ctx, true, func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		if err := Sign(req, c.credential, c.now().UTC()); err != nil {
			return nil, err
		}
		return c.httpClient.Do(req)
	})
	if err != nil {
		return nil, err
	}
	if httpResp == nil {
		return nil, fmt.Errorf("tencent cos client: empty response")
	}
	defer closeResponse(httpResp)

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read tencent cos response: %w", err)
	}
	if err := decodeError(httpResp, body); err != nil {
		return nil, err
	}

	var out ListBucketsResponse
	if len(body) == 0 {
		return &out, nil
	}
	if err := xml.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode tencent cos response: %w", err)
	}
	return &out, nil
}

func (c *Client) serviceURL() (*url.URL, error) {
	rawURL := strings.TrimSpace(c.serviceEndpoint)
	if rawURL == "" {
		rawURL = defaultServiceEndpoint
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("tencent cos client: invalid service endpoint %q: %w", rawURL, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("tencent cos client: invalid service endpoint %q", rawURL)
	}
	if strings.TrimSpace(u.Path) == "" {
		u.Path = "/"
	}
	return u, nil
}

func closeResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 8<<10))
	_ = resp.Body.Close()
}
