package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/internal/httpclient"
)

func TestRetryPolicyNonIdempotentDoesNotRetry(t *testing.T) {
	var attempts int
	policy := retryPolicy{
		MaxAttempts: 3,
		BaseDelay:  500 * time.Millisecond,
		Sleep:      func(context.Context, time.Duration) error { return nil },
		Rand:       func() float64 { return 0 },
		Clock:      func() time.Time { return time.Unix(1700000000, 0) },
		Classifier: gcpRetryClassifier,
	}

	resp, err := policy.Do(context.Background(), false, func() (*http.Response, error) {
		attempts++
		return newHTTPResponse(http.StatusInternalServerError, nil, `server error`), nil
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	if attempts != 1 {
		t.Fatalf("expected one attempt, got %d", attempts)
	}
}

func TestRetryPolicyIdempotentRetriesTwice(t *testing.T) {
	var attempts int
	var slept []time.Duration
	policy := retryPolicy{
		MaxAttempts: 3,
		BaseDelay: 500 * time.Millisecond,
		Sleep: func(_ context.Context, d time.Duration) error {
			slept = append(slept, d)
			return nil
		},
		Rand:       func() float64 { return 0 },
		Clock:      func() time.Time { return time.Unix(1700000000, 0) },
		Classifier: gcpRetryClassifier,
	}

	resp, err := policy.Do(context.Background(), true, func() (*http.Response, error) {
		attempts++
		if attempts < 3 {
			return newHTTPResponse(http.StatusInternalServerError, nil, `server error`), nil
		}
		return newHTTPResponse(http.StatusOK, nil, `{}`), nil
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	if attempts != 3 {
		t.Fatalf("expected three attempts, got %d", attempts)
	}
	if len(slept) != 2 || slept[0] != 250*time.Millisecond || slept[1] != 500*time.Millisecond {
		t.Fatalf("unexpected backoff sleeps: %v", slept)
	}
}

func TestRetryPolicyUsesRetryAfterHeader(t *testing.T) {
	var attempts int
	var slept []time.Duration
	policy := retryPolicy{
		MaxAttempts: 3,
		BaseDelay: 500 * time.Millisecond,
		Sleep: func(_ context.Context, d time.Duration) error {
			slept = append(slept, d)
			return nil
		},
		Rand:       func() float64 { return 0 },
		Clock:      func() time.Time { return time.Unix(1700000000, 0) },
		Classifier: gcpRetryClassifier,
	}

	_, err := policy.Do(context.Background(), true, func() (*http.Response, error) {
		attempts++
		if attempts == 1 {
			return newHTTPResponse(http.StatusTooManyRequests, http.Header{"Retry-After": {"2"}}, `{"error":{"code":429,"status":"RESOURCE_EXHAUSTED"}}`), nil
		}
		return newHTTPResponse(http.StatusOK, nil, `{}`), nil
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected two attempts, got %d", attempts)
	}
	if len(slept) != 1 || slept[0] != 2*time.Second {
		t.Fatalf("unexpected sleeps: %v", slept)
	}
}

func TestRetryPolicyContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	policy := retryPolicy{
		MaxAttempts: 3,
		BaseDelay:  500 * time.Millisecond,
		Sleep:      httpclient.SleepWithContext,
		Rand:       func() float64 { return 0 },
		Clock:      func() time.Time { return time.Unix(1700000000, 0) },
		Classifier: gcpRetryClassifier,
	}

	_, err := policy.Do(ctx, true, func() (*http.Response, error) {
		return nil, errors.New("network down")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func newHTTPResponse(status int, headers http.Header, body string) *http.Response {
	if headers == nil {
		headers = http.Header{}
	}
	return &http.Response{
		StatusCode: status,
		Header:     headers,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
