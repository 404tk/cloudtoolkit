package oss

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
)

const (
	defaultRegion               = "cn-hangzhou"
	defaultServiceEndpointFmt   = "https://oss-%s.aliyuncs.com"
	defaultBucketEndpointFormat = "https://%s.oss-%s.aliyuncs.com"
)

type Option func(*Client)

type Client struct {
	credential      aliauth.Credential
	httpClient      *http.Client
	retryPolicy     api.RetryPolicy
	now             func() time.Time
	serviceEndpoint string
}

func NewClient(cred aliauth.Credential, opts ...Option) *Client {
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

func WithRetryPolicy(policy api.RetryPolicy) Option {
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

func WithServiceEndpoint(rawURL string) Option {
	return func(c *Client) {
		c.serviceEndpoint = strings.TrimSpace(rawURL)
	}
}

func (c *Client) ListBuckets(ctx context.Context, region string) (*ListBucketsResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.credential.Validate(); err != nil {
		return nil, err
	}

	u, err := c.serviceURL(region)
	if err != nil {
		return nil, err
	}
	query := u.Query()
	query.Set("max-keys", "1000")
	u.RawQuery = query.Encode()

	httpResp, err := c.retryPolicy.Do(ctx, true, func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		if err := Sign(req, c.credential, "", c.now().UTC()); err != nil {
			return nil, err
		}
		return c.httpClient.Do(req)
	})
	if err != nil {
		return nil, err
	}
	if httpResp == nil {
		return nil, fmt.Errorf("alibaba oss client: empty response")
	}
	defer closeResponse(httpResp)

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read alibaba oss response: %w", err)
	}
	if err := decodeError(httpResp, body); err != nil {
		return nil, err
	}

	var out ListBucketsResponse
	if len(body) == 0 {
		return &out, nil
	}
	if err := xml.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode alibaba oss response: %w", err)
	}
	return &out, nil
}

func (c *Client) ListObjectsV2(ctx context.Context, bucket, region, continuationToken string, maxKeys int) (ListObjectsResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.credential.Validate(); err != nil {
		return ListObjectsResponse{}, err
	}
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return ListObjectsResponse{}, fmt.Errorf("alibaba oss client: empty bucket")
	}
	region = strings.TrimSpace(region)
	if region == "" || region == "all" {
		return ListObjectsResponse{}, fmt.Errorf("alibaba oss client: empty region")
	}
	if maxKeys <= 0 {
		maxKeys = 1000
	}

	u, err := c.bucketURL(bucket, region)
	if err != nil {
		return ListObjectsResponse{}, err
	}
	query := u.Query()
	query.Set("list-type", "2")
	query.Set("encoding-type", "url")
	query.Set("max-keys", fmt.Sprintf("%d", maxKeys))
	if continuationToken = strings.TrimSpace(continuationToken); continuationToken != "" {
		query.Set("continuation-token", continuationToken)
	}
	u.RawQuery = query.Encode()

	httpResp, err := c.retryPolicy.Do(ctx, true, func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		if err := Sign(req, c.credential, bucket, c.now().UTC()); err != nil {
			return nil, err
		}
		return c.httpClient.Do(req)
	})
	if err != nil {
		return ListObjectsResponse{}, err
	}
	if httpResp == nil {
		return ListObjectsResponse{}, fmt.Errorf("alibaba oss client: empty response")
	}
	defer closeResponse(httpResp)

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return ListObjectsResponse{}, fmt.Errorf("read alibaba oss response: %w", err)
	}
	if err := decodeError(httpResp, body); err != nil {
		return ListObjectsResponse{}, err
	}

	var out ListObjectsResponse
	if len(body) == 0 {
		return out, nil
	}
	if err := xml.Unmarshal(body, &out); err != nil {
		return ListObjectsResponse{}, fmt.Errorf("decode alibaba oss response: %w", err)
	}
	if err := decodeListObjectsResponse(&out); err != nil {
		return ListObjectsResponse{}, err
	}
	return out, nil
}

func (c *Client) serviceURL(region string) (*url.URL, error) {
	rawURL := c.serviceEndpoint
	if rawURL == "" {
		rawURL = fmt.Sprintf(defaultServiceEndpointFmt, normalizeServiceRegion(region))
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("alibaba oss client: invalid service endpoint %q: %w", rawURL, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("alibaba oss client: invalid service endpoint %q", rawURL)
	}
	if strings.TrimSpace(u.Path) == "" {
		u.Path = "/"
	}
	return u, nil
}

func (c *Client) bucketURL(bucket, region string) (*url.URL, error) {
	rawURL := fmt.Sprintf(defaultBucketEndpointFormat, bucket, region)
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("alibaba oss client: invalid bucket endpoint %q: %w", rawURL, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("alibaba oss client: invalid bucket endpoint %q", rawURL)
	}
	if strings.TrimSpace(u.Path) == "" {
		u.Path = "/"
	}
	return u, nil
}

func normalizeServiceRegion(region string) string {
	region = strings.TrimSpace(region)
	if region == "" || region == "all" {
		return defaultRegion
	}
	return region
}

func closeResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 8<<10))
	_ = resp.Body.Close()
}
