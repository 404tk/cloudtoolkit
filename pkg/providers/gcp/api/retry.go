package api

import (
	"errors"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/internal/httpclient"
)

type RetryPolicy = httpclient.RetryPolicy
type retryPolicy = httpclient.RetryPolicy

func DefaultRetryPolicy() RetryPolicy {
	return retryPolicy{
		MaxAttempts: 3,
		BaseDelay:   500 * time.Millisecond,
		Sleep:       httpclient.SleepWithContext,
		Rand:        rand.Float64,
		Clock:       time.Now,
		Classifier:  gcpRetryClassifier,
	}
}

func gcpRetryClassifier(policy httpclient.RetryPolicy, resp *http.Response, err error, attempt int) httpclient.RetryDecision {
	if err != nil {
		return httpclient.RetryDecision{Retry: true, Delay: policy.JitterBackoff(attempt)}
	}
	if resp == nil {
		return httpclient.RetryDecision{}
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
		if delay, ok := httpclient.ParseRetryAfter(resp.Header.Get("Retry-After"), policy.Now()); ok {
			return httpclient.RetryDecision{Retry: true, Delay: delay}
		}
		return httpclient.RetryDecision{Retry: true, Delay: policy.JitterBackoff(attempt)}
	}
	if resp.StatusCode < http.StatusBadRequest {
		return httpclient.RetryDecision{}
	}

	body, readErr := httpclient.SnapshotBody(resp)
	if readErr != nil {
		return httpclient.RetryDecision{Err: readErr}
	}

	apiErr := &APIError{}
	if err := DecodeError(resp.StatusCode, body); errors.As(err, &apiErr) && strings.EqualFold(apiErr.Status, "RESOURCE_EXHAUSTED") {
		if delay, ok := httpclient.ParseRetryAfter(resp.Header.Get("Retry-After"), policy.Now()); ok {
			return httpclient.RetryDecision{Retry: true, Delay: delay}
		}
		return httpclient.RetryDecision{Retry: true, Delay: policy.JitterBackoff(attempt)}
	}

	return httpclient.RetryDecision{}
}
