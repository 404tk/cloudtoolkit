package api

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"time"
)

type RetryPolicy interface {
	Do(ctx context.Context, idempotent bool, fn func() (*http.Response, error)) (*http.Response, error)
}

type retryPolicy struct {
	baseDelay time.Duration
	sleep     func(context.Context, time.Duration) error
	rand      func() float64
}

func DefaultRetryPolicy() RetryPolicy {
	return retryPolicy{
		baseDelay: 200 * time.Millisecond,
		sleep:     sleepWithContext,
		rand:      rand.Float64,
	}
}

func (p retryPolicy) Do(
	ctx context.Context,
	idempotent bool,
	fn func() (*http.Response, error),
) (*http.Response, error) {
	attempts := 1
	if idempotent {
		attempts = 2
	}
	sleepFn := p.sleep
	if sleepFn == nil {
		sleepFn = sleepWithContext
	}

	var (
		resp *http.Response
		err  error
	)
	for attempt := 1; attempt <= attempts; attempt++ {
		if ctx.Err() != nil {
			closeResponse(resp)
			return nil, ctx.Err()
		}
		resp, err = fn()
		if !shouldRetry(resp, err) || attempt == attempts {
			return resp, err
		}
		closeResponse(resp)
		if err := sleepFn(ctx, p.backoff(attempt)); err != nil {
			return nil, err
		}
	}
	return resp, err
}

func (p retryPolicy) backoff(attempt int) time.Duration {
	if p.baseDelay <= 0 {
		return 0
	}
	jitter := 1.0
	if p.rand != nil {
		jitter = 0.5 + p.rand()
	}
	multiplier := time.Duration(1 << uint(attempt-1))
	return time.Duration(float64(p.baseDelay*multiplier) * jitter)
}

func shouldRetry(resp *http.Response, err error) bool {
	if err != nil {
		return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
	}
	if resp == nil {
		return false
	}
	return resp.StatusCode >= http.StatusInternalServerError
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func closeResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 8<<10))
	_ = resp.Body.Close()
}
