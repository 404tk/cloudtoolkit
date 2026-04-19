package api

import (
	"context"
	"net/url"
)

// Pager walks Azure ARM list responses that use nextLink.
type Pager[T any] struct {
	client *Client
	req    Request
}

func NewPager[T any](c *Client, initial Request) *Pager[T] {
	return &Pager[T]{
		client: c,
		req:    initial,
	}
}

func (p *Pager[T]) All(ctx context.Context) ([]T, error) {
	type page struct {
		Value    []T    `json:"value"`
		NextLink string `json:"nextLink"`
	}

	req := Request{
		Method:     p.req.Method,
		Path:       p.req.Path,
		Query:      cloneValues(p.req.Query),
		Headers:    cloneHeader(p.req.Headers),
		Body:       append([]byte(nil), p.req.Body...),
		Idempotent: p.req.Idempotent,
	}

	items := make([]T, 0)
	for {
		var current page
		if err := p.client.Do(ctx, req, &current); err != nil {
			return nil, err
		}
		items = append(items, current.Value...)
		if current.NextLink == "" {
			return items, nil
		}
		nextURL, err := url.Parse(current.NextLink)
		if err != nil {
			return nil, err
		}
		req.Path = nextURL.Path
		req.Query = nextURL.Query()
	}
}
