package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestRetryPolicyRetriesIdempotentRequests(t *testing.T) {
	attempts := 0
	policy := RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		Sleep:       func(context.Context, time.Duration) error { return nil },
	}
	resp, err := policy.Do(context.Background(), true, func() (*http.Response, error) {
		attempts++
		if attempts == 1 {
			return &http.Response{StatusCode: http.StatusBadGateway, Body: http.NoBody}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if resp.StatusCode != http.StatusOK || attempts != 2 {
		t.Fatalf("unexpected result: status=%d attempts=%d", resp.StatusCode, attempts)
	}
}

func TestRetryPolicyDoesNotRetryNonIdempotentRequests(t *testing.T) {
	attempts := 0
	policy := RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		Sleep:       func(context.Context, time.Duration) error { return nil },
	}
	_, err := policy.Do(context.Background(), false, func() (*http.Response, error) {
		attempts++
		return nil, errors.New("boom")
	})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 1 {
		t.Fatalf("unexpected attempts: %d", attempts)
	}
}
