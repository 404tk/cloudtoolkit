package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestRetryPolicyRetriesIdempotentRequests(t *testing.T) {
	var attempts int
	policy := RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    time.Millisecond,
		Sleep:       func(context.Context, time.Duration) error { return nil },
	}

	resp, err := policy.Do(context.Background(), true, func() (*http.Response, error) {
		attempts++
		if attempts < 3 {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("retry")),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
		}, nil
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if attempts != 3 {
		t.Fatalf("unexpected attempts: %d", attempts)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}

func TestRetryPolicyDoesNotRetryNonIdempotentRequests(t *testing.T) {
	var attempts int
	policy := RetryPolicy{
		MaxAttempts: 3,
		Sleep:       func(context.Context, time.Duration) error { return nil },
	}

	resp, err := policy.Do(context.Background(), false, func() (*http.Response, error) {
		attempts++
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader("no-retry")),
		}, nil
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if attempts != 1 {
		t.Fatalf("unexpected attempts: %d", attempts)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}

func TestRetryPolicyStopsOnContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	policy := RetryPolicy{
		MaxAttempts: 3,
		Sleep:       func(context.Context, time.Duration) error { return nil },
	}
	_, err := policy.Do(ctx, true, func() (*http.Response, error) {
		return nil, errors.New("should not be called")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
