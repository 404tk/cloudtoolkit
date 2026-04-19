package obs

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/endpoint"
)

type Option func(*Client)

type Client struct {
	credential  auth.Credential
	httpClient  *http.Client
	retryPolicy api.RetryPolicy
	now         func() time.Time
}

func NewClient(cred auth.Credential, opts ...Option) *Client {
	client := &Client{
		credential:  cred,
		httpClient:  api.NewHTTPClient(),
		retryPolicy: api.DefaultRetryPolicy(),
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

func WithRetryPolicy(p api.RetryPolicy) Option {
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

func (c *Client) ListBuckets(ctx context.Context, endpointRegion string) (*ListBucketsResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.credential.Validate(); err != nil {
		return nil, err
	}
	endpointRegion = strings.TrimSpace(endpointRegion)
	if endpointRegion == "" {
		return nil, fmt.Errorf("huawei obs client: empty region")
	}

	rawURL := endpoint.For("obs", endpointRegion, c.credential.Intl)
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("huawei obs client: invalid endpoint %q: %w", rawURL, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("huawei obs client: invalid endpoint %q", rawURL)
	}
	if strings.TrimSpace(u.Path) == "" {
		u.Path = "/"
	}

	signed, err := Sign(&SignRequest{
		Method:    http.MethodGet,
		Path:      u.Path,
		AccessKey: c.credential.AK,
		SecretKey: c.credential.SK,
		Timestamp: c.now().UTC(),
	})
	if err != nil {
		return nil, err
	}

	httpResp, err := c.retryPolicy.Do(ctx, true, func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Host = u.Host
		req.Header = signed.Clone()
		return c.httpClient.Do(req)
	})
	if err != nil {
		return nil, err
	}
	if httpResp == nil {
		return nil, fmt.Errorf("huawei obs client: empty response")
	}
	defer closeResponse(httpResp)

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read huawei obs response: %w", err)
	}
	if err := decodeError(httpResp.StatusCode, httpResp.Header, body); err != nil {
		return nil, err
	}

	var out ListBucketsResponse
	if len(body) == 0 {
		return &out, nil
	}
	if err := xml.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode huawei obs response: %w", err)
	}
	return &out, nil
}

func (c *Client) ListObjects(ctx context.Context, bucket, region, marker string, maxKeys int) (ListObjectsResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.credential.Validate(); err != nil {
		return ListObjectsResponse{}, err
	}
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return ListObjectsResponse{}, fmt.Errorf("huawei obs client: empty bucket")
	}
	region = strings.TrimSpace(region)
	if region == "" {
		return ListObjectsResponse{}, fmt.Errorf("huawei obs client: empty region")
	}
	if maxKeys <= 0 {
		maxKeys = 1000
	}

	rawURL := endpoint.For("obs", region, c.credential.Intl)
	u, err := url.Parse(rawURL)
	if err != nil {
		return ListObjectsResponse{}, fmt.Errorf("huawei obs client: invalid endpoint %q: %w", rawURL, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return ListObjectsResponse{}, fmt.Errorf("huawei obs client: invalid endpoint %q", rawURL)
	}

	u.Path = "/" + bucket
	query := url.Values{}
	query.Set("max-keys", fmt.Sprintf("%d", maxKeys))
	if strings.TrimSpace(marker) != "" {
		query.Set("marker", marker)
	}
	u.RawQuery = query.Encode()

	signed, err := Sign(&SignRequest{
		Method:    http.MethodGet,
		Path:      u.Path,
		Query:     query,
		Scheme:    authSchemeV2,
		AccessKey: c.credential.AK,
		SecretKey: c.credential.SK,
		Timestamp: c.now().UTC(),
	})
	if err != nil {
		return ListObjectsResponse{}, err
	}

	httpResp, err := c.retryPolicy.Do(ctx, true, func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Host = u.Host
		req.Header = signed.Clone()
		return c.httpClient.Do(req)
	})
	if err != nil {
		return ListObjectsResponse{}, err
	}
	if httpResp == nil {
		return ListObjectsResponse{}, fmt.Errorf("huawei obs client: empty response")
	}
	defer closeResponse(httpResp)

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return ListObjectsResponse{}, fmt.Errorf("read huawei obs response: %w", err)
	}
	if err := decodeError(httpResp.StatusCode, httpResp.Header, body); err != nil {
		return ListObjectsResponse{}, err
	}

	var out ListObjectsResponse
	if len(body) == 0 {
		return out, nil
	}
	if err := xml.Unmarshal(body, &out); err != nil {
		return ListObjectsResponse{}, fmt.Errorf("decode huawei obs response: %w", err)
	}
	return out, nil
}

func closeResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 8<<10))
	_ = resp.Body.Close()
}
