package api

import (
	"context"
	"errors"
	"math/rand"
	"net"
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
}

func DefaultRetryPolicy() RetryPolicy {
	return retryPolicy{
		baseDelay: 300 * time.Millisecond,
		sleep:     sleepWithContext,
		rand:      rand.Float64,
	}
}

func (p retryPolicy) Do(ctx context.Context, idempotent bool, fn func() (*http.Response, error)) (*http.Response, error) {
	attempts := 1
	if idempotent {
		attempts = 3
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
		delay := p.retryDelay(resp, attempt)
		closeResponse(resp)
		if err := sleepFn(ctx, delay); err != nil {
			return nil, err
		}
	}
	return resp, err
}

func (p retryPolicy) retryDelay(resp *http.Response, attempt int) time.Duration {
	if resp != nil {
		if retryAfter := parseRetryAfter(resp.Header.Get("Retry-After")); retryAfter > 0 {
			return retryAfter
		}
	}
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
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false
		}
		var netErr net.Error
		return errors.As(err, &netErr) || !strings.Contains(strings.ToLower(err.Error()), "tls:")
	}
	if resp == nil {
		return false
	}
	return resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError
}

func parseRetryAfter(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(value); err == nil {
		if delay := time.Until(when); delay > 0 {
			return delay
		}
	}
	return 0
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
