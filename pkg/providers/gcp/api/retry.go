package api

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type RetryPolicy interface {
	Do(ctx context.Context, idempotent bool, fn func() (*http.Response, error)) (*http.Response, error)
}

type retryPolicy struct {
	baseDelay time.Duration
	sleep     func(context.Context, time.Duration) error
	rand      func() float64
	clock     func() time.Time
}

func DefaultRetryPolicy() RetryPolicy {
	return retryPolicy{
		baseDelay: 500 * time.Millisecond,
		sleep:     sleepContext,
		rand:      rand.Float64,
		clock:     time.Now,
	}
}

func (p retryPolicy) Do(ctx context.Context, idempotent bool, fn func() (*http.Response, error)) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	maxAttempts := 1
	if idempotent {
		maxAttempts = 3
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		resp, err := fn()
		if !idempotent || attempt == maxAttempts-1 {
			if err != nil {
				return nil, err
			}
			return resp, nil
		}

		shouldRetry, delay, inspectErr := p.shouldRetry(resp, err, attempt)
		if resp != nil && !shouldRetry {
			return resp, err
		}
		if inspectErr != nil {
			return nil, inspectErr
		}
		if !shouldRetry {
			if err != nil {
				return nil, err
			}
			return resp, nil
		}

		lastErr = err
		if resp != nil {
			closeResponse(resp)
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := p.sleepFor(ctx, delay); err != nil {
			if lastErr != nil {
				return nil, lastErr
			}
			return nil, err
		}
	}
	return nil, lastErr
}

func (p retryPolicy) shouldRetry(resp *http.Response, err error, attempt int) (bool, time.Duration, error) {
	if err != nil {
		return true, p.backoff(attempt), nil
	}
	if resp == nil {
		return false, 0, nil
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
		return true, p.retryDelay(resp, attempt), nil
	}

	if resp.StatusCode < http.StatusBadRequest {
		return false, 0, nil
	}

	body, err := snapshotBody(resp)
	if err != nil {
		return false, 0, err
	}
	apiErr := &APIError{}
	if err := DecodeError(resp.StatusCode, body); errors.As(err, &apiErr) && strings.EqualFold(apiErr.Status, "RESOURCE_EXHAUSTED") {
		return true, p.retryDelay(resp, attempt), nil
	}

	return false, 0, nil
}

func (p retryPolicy) retryDelay(resp *http.Response, attempt int) time.Duration {
	if resp != nil {
		if delay, ok := parseRetryAfter(resp.Header.Get("Retry-After"), p.now()); ok {
			return delay
		}
	}
	return p.backoff(attempt)
}

func (p retryPolicy) backoff(attempt int) time.Duration {
	base := p.baseDelay
	if base <= 0 {
		base = 500 * time.Millisecond
	}
	factor := 0.5
	if p.rand != nil {
		factor += p.rand()
	}
	return time.Duration(float64(base<<attempt) * factor)
}

func (p retryPolicy) sleepFor(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	if p.sleep != nil {
		return p.sleep(ctx, d)
	}
	return sleepContext(ctx, d)
}

func (p retryPolicy) now() time.Time {
	if p.clock != nil {
		return p.clock()
	}
	return time.Now()
}

func parseRetryAfter(value string, now time.Time) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds < 0 {
			seconds = 0
		}
		return time.Duration(seconds) * time.Second, true
	}
	when, err := time.Parse(http.TimeFormat, value)
	if err != nil {
		return 0, false
	}
	if when.Before(now) {
		return 0, true
	}
	return when.Sub(now), true
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func snapshotBody(resp *http.Response) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}
