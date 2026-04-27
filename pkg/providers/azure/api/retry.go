package api

import (
	"context"
	"errors"
	"math/rand"
	"net"
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
		BaseDelay:   300 * time.Millisecond,
		Sleep:       httpclient.SleepWithContext,
		Rand:        rand.Float64,
		Clock:       time.Now,
		Classifier:  azureRetryClassifier,
	}
}

func azureRetryClassifier(policy httpclient.RetryPolicy, resp *http.Response, err error, attempt int) httpclient.RetryDecision {
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return httpclient.RetryDecision{}
		}
		var netErr net.Error
		if errors.As(err, &netErr) || !strings.Contains(strings.ToLower(err.Error()), "tls:") {
			return httpclient.RetryDecision{Retry: true, Delay: policy.JitterBackoff(attempt)}
		}
		return httpclient.RetryDecision{}
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
	return httpclient.RetryDecision{}
}
