package httpclient

import (
	"context"
	"errors"
	"net/http"
	"time"
)

type RetryDecision struct {
	Retry bool
	Delay time.Duration
	Err   error
}

type RetryClassifier func(policy RetryPolicy, resp *http.Response, err error, attempt int) RetryDecision

type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Sleep       func(context.Context, time.Duration) error
	Rand        func() float64
	Clock       func() time.Time
	Classifier  RetryClassifier
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   200 * time.Millisecond,
		MaxDelay:    2 * time.Second,
		Sleep:       SleepWithContext,
		Classifier:  DefaultRetryClassifier,
	}
}

func DefaultRetryClassifier(policy RetryPolicy, resp *http.Response, err error, attempt int) RetryDecision {
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return RetryDecision{}
		}
		return RetryDecision{Retry: true, Delay: policy.ExponentialBackoff(attempt)}
	}
	if resp == nil {
		return RetryDecision{}
	}
	switch resp.StatusCode {
	case http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return RetryDecision{Retry: true, Delay: policy.ExponentialBackoff(attempt)}
	default:
		return RetryDecision{}
	}
}

func (p RetryPolicy) Do(ctx context.Context, idempotent bool, fn func() (*http.Response, error)) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	attempts := p.MaxAttempts
	if attempts <= 0 {
		attempts = 1
	}
	if !idempotent {
		attempts = 1
	}

	sleepFn := p.Sleep
	if sleepFn == nil {
		sleepFn = SleepWithContext
	}

	var (
		resp *http.Response
		err  error
	)
	for attempt := 1; attempt <= attempts; attempt++ {
		if ctx.Err() != nil {
			CloseResponse(resp)
			return nil, ctx.Err()
		}

		resp, err = fn()
		if attempt == attempts {
			return resp, err
		}

		classifier := p.Classifier
		if classifier == nil {
			classifier = DefaultRetryClassifier
		}

		decision := classifier(p, resp, err, attempt)
		if decision.Err != nil {
			CloseResponse(resp)
			return nil, decision.Err
		}
		if !decision.Retry {
			return resp, err
		}

		CloseResponse(resp)
		if err := sleepFn(ctx, decision.Delay); err != nil {
			return nil, err
		}
	}

	return resp, err
}

func (p RetryPolicy) ExponentialBackoff(attempt int) time.Duration {
	if p.BaseDelay <= 0 || attempt <= 0 {
		return 0
	}
	delay := p.BaseDelay * time.Duration(1<<uint(attempt-1))
	if p.MaxDelay > 0 && delay > p.MaxDelay {
		return p.MaxDelay
	}
	return delay
}

func (p RetryPolicy) JitterBackoff(attempt int) time.Duration {
	if p.BaseDelay <= 0 || attempt <= 0 {
		return 0
	}
	factor := 1.0
	if p.Rand != nil {
		factor = 0.5 + p.Rand()
	}
	delay := time.Duration(float64(p.BaseDelay*time.Duration(1<<uint(attempt-1))) * factor)
	if p.MaxDelay > 0 && delay > p.MaxDelay {
		return p.MaxDelay
	}
	return delay
}

func (p RetryPolicy) Now() time.Time {
	if p.Clock != nil {
		return p.Clock()
	}
	return time.Now()
}
