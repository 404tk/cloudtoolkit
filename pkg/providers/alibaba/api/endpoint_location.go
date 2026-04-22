package api

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const locationReadonlyEndpoint = "location-readonly.aliyuncs.com"

type endpointCache struct {
	mu    sync.RWMutex
	items map[string]string
}

func (c *endpointCache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, ok := c.items[key]
	return value, ok
}

func (c *endpointCache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.items == nil {
		c.items = map[string]string{}
	}
	c.items[key] = value
}

var locationEndpointCache = &endpointCache{items: map[string]string{}}

type describeLocationEndpointsResponse struct {
	Endpoints locationEndpoints `json:"Endpoints"`
	RequestID string            `json:"RequestId"`
	Success   bool              `json:"Success"`
}

type locationEndpoints struct {
	Endpoint []locationEndpoint `json:"Endpoint"`
}

type locationEndpoint struct {
	Endpoint string `json:"Endpoint"`
}

func (c *Client) resolveEndpointByLocation(ctx context.Context, product, region string) string {
	serviceCode, ok := locationServiceCode(product)
	if !ok {
		return ""
	}
	cacheKey := strings.ToLower(strings.TrimSpace(product)) + "#" + strings.ToLower(strings.TrimSpace(region))
	if cached, ok := locationEndpointCache.Get(cacheKey); ok && cached != "" {
		return cached
	}

	query := url.Values{}
	query.Set("Id", NormalizeRegion(region))
	query.Set("ServiceCode", serviceCode)
	query.Set("Type", "openAPI")

	var resp describeLocationEndpointsResponse
	if err := c.Do(ctx, Request{
		Product:      "Location",
		Version:      "2015-06-12",
		Action:       "DescribeEndpoints",
		Method:       http.MethodGet,
		Query:        query,
		Host:         locationReadonlyEndpoint,
		Idempotent:   true,
		SkipRegionID: true,
	}, &resp); err != nil {
		return ""
	}
	for _, endpoint := range resp.Endpoints.Endpoint {
		host := strings.TrimSpace(endpoint.Endpoint)
		if host == "" {
			continue
		}
		locationEndpointCache.Set(cacheKey, host)
		return host
	}
	return ""
}

func locationServiceCode(product string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(product)) {
	case "ecs":
		return "ecs", true
	default:
		return "", false
	}
}
