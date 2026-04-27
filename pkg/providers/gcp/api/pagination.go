package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/404tk/cloudtoolkit/pkg/providers/internal/httpclient"
)

type Pager[T any] struct {
	client     *Client
	req        Request
	itemsField string
}

func NewPager[T any](c *Client, initial Request, itemsField string) *Pager[T] {
	return &Pager[T]{
		client:     c,
		req:        initial,
		itemsField: itemsField,
	}
}

func (p *Pager[T]) All(ctx context.Context) ([]T, error) {
	req := Request{
		Method:     p.req.Method,
		BaseURL:    p.req.BaseURL,
		Path:       p.req.Path,
		Query:      httpclient.CloneValues(p.req.Query),
		Headers:    httpclient.CloneHeader(p.req.Headers),
		Body:       append([]byte(nil), p.req.Body...),
		Idempotent: p.req.Idempotent,
	}
	if req.Query == nil {
		req.Query = url.Values{}
	}
	if req.Query.Get("maxResults") == "" && req.Query.Get("pageSize") == "" {
		req.Query.Set("maxResults", "500")
	}

	items := make([]T, 0)
	for {
		var raw map[string]json.RawMessage
		if err := p.client.Do(ctx, req, &raw); err != nil {
			return nil, err
		}

		if part, ok := raw[p.itemsField]; ok && len(part) > 0 && string(part) != "null" {
			var pageItems []T
			if err := json.Unmarshal(part, &pageItems); err != nil {
				return nil, fmt.Errorf("decode gcp page items %q: %w", p.itemsField, err)
			}
			items = append(items, pageItems...)
		}

		var nextToken string
		if part, ok := raw["nextPageToken"]; ok && len(part) > 0 && string(part) != "null" {
			if err := json.Unmarshal(part, &nextToken); err != nil {
				return nil, fmt.Errorf("decode gcp nextPageToken: %w", err)
			}
		}
		if nextToken == "" {
			return items, nil
		}
		req.Query.Set("pageToken", nextToken)
	}
}
