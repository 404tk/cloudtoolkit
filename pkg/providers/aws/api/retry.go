package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"
)

type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Sleep       func(context.Context, time.Duration) error
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   200 * time.Millisecond,
		MaxDelay:    2 * time.Second,
		Sleep:       sleepWithContext,
	}
}

func (p RetryPolicy) Do(
	ctx context.Context,
	idempotent bool,
	fn func() (*http.Response, error),
) (*http.Response, error) {
	attempts := p.MaxAttempts
	if attempts <= 0 {
		attempts = 1
	}
	if !idempotent {
		attempts = 1
	}
	sleepFn := p.Sleep
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

func (p RetryPolicy) backoff(attempt int) time.Duration {
	if p.BaseDelay <= 0 {
		return 0
	}
	delay := p.BaseDelay * time.Duration(1<<uint(attempt-1))
	if p.MaxDelay > 0 && delay > p.MaxDelay {
		return p.MaxDelay
	}
	return delay
}

func shouldRetry(resp *http.Response, err error) bool {
	if err != nil {
		return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
	}
	if resp == nil {
		return false
	}
	switch resp.StatusCode {
	case http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
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
