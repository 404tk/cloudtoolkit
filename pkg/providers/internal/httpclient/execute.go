package httpclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

func Execute(
	ctx context.Context,
	client *http.Client,
	retry RetryPolicy,
	template *http.Request,
	idempotent bool,
) (*http.Response, []byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if client == nil {
		client = http.DefaultClient
	}
	if template == nil {
		return nil, nil, fmt.Errorf("httpclient: nil request")
	}

	body, err := snapshotRequestBody(template)
	if err != nil {
		return nil, nil, err
	}

	resp, err := retry.Do(ctx, idempotent, func() (*http.Response, error) {
		req, err := cloneRequest(ctx, template, body)
		if err != nil {
			return nil, err
		}
		return client.Do(req)
	})
	if err != nil || resp == nil {
		if err == nil && resp == nil {
			err = fmt.Errorf("httpclient: nil response")
		}
		return resp, nil, err
	}

	respBody, err := SnapshotBody(resp)
	if err != nil {
		CloseResponse(resp)
		return nil, nil, err
	}

	return resp, respBody, nil
}

func snapshotRequestBody(req *http.Request) ([]byte, error) {
	if req == nil || req.Body == nil {
		return nil, nil
	}
	if req.GetBody != nil {
		rc, err := req.GetBody()
		if err != nil {
			return nil, fmt.Errorf("httpclient: get request body: %w", err)
		}
		defer rc.Close()

		body, err := io.ReadAll(rc)
		if err != nil {
			return nil, fmt.Errorf("httpclient: read request body: %w", err)
		}
		return body, nil
	}

	body, err := io.ReadAll(req.Body)
	_ = req.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("httpclient: read request body: %w", err)
	}

	req.Body = io.NopCloser(bytes.NewReader(body))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	req.ContentLength = int64(len(body))
	return body, nil
}

func cloneRequest(ctx context.Context, template *http.Request, body []byte) (*http.Request, error) {
	req := template.Clone(ctx)
	if body == nil {
		req.Body = nil
		req.GetBody = nil
		req.ContentLength = 0
		return req, nil
	}

	req.Body = io.NopCloser(bytes.NewReader(body))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	req.ContentLength = int64(len(body))
	return req, nil
}
