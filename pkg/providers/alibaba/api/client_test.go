package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
)

func TestClientGetCallerIdentity(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		query := r.URL.Query()
		if got := query.Get("Action"); got != "GetCallerIdentity" {
			t.Fatalf("unexpected action: %s", got)
		}
		if got := query.Get("Version"); got != "2015-04-01" {
			t.Fatalf("unexpected version: %s", got)
		}
		if got := query.Get("RegionId"); got != DefaultRegion {
			t.Fatalf("unexpected region: %s", got)
		}
		if got := query.Get("SecurityToken"); got != "token" {
			t.Fatalf("unexpected security token: %s", got)
		}
		if got := query.Get("Signature"); got == "" {
			t.Fatalf("missing signature")
		}
		if got := r.Header.Get("User-Agent"); got != "ctk" {
			t.Fatalf("unexpected user-agent: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"Arn":"acs:ram::1234567890123456:root","RequestId":"req-1"}`)
	}))
	defer server.Close()

	client := NewClient(
		auth.New("ak", "sk", "token"),
		WithBaseURL(server.URL),
		WithClock(func() time.Time { return time.Unix(1713376800, 0).UTC() }),
		WithNonce(func() string { return "nonce" }),
	)
	resp, err := client.GetCallerIdentity(context.Background(), "all")
	if err != nil {
		t.Fatalf("get caller identity: %v", err)
	}
	if got, want := resp.Arn, "acs:ram::1234567890123456:root"; got != want {
		t.Fatalf("unexpected arn: got %q want %q", got, want)
	}
}

func TestClientRetriesAndDecodesBSSFailure(t *testing.T) {
	t.Parallel()

	var attempts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			http.Error(w, "retry later", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"Code":"Forbidden","Message":"denied","RequestId":"req-2","Success":false}`)
	}))
	defer server.Close()

	client := NewClient(
		auth.New("ak", "sk", ""),
		WithBaseURL(server.URL),
		WithClock(func() time.Time { return time.Unix(1713376800, 0).UTC() }),
		WithNonce(func() string { return "nonce" }),
		WithRetryPolicy(RetryPolicy{
			MaxAttempts: 2,
			Sleep: func(context.Context, time.Duration) error {
				return nil
			},
		}),
	)

	_, err := client.QueryAccountBalance(context.Background(), DefaultRegion)
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("unexpected error type: %T %v", err, err)
	}
	if apiErr.Code != "Forbidden" || apiErr.Message != "denied" {
		t.Fatalf("unexpected api error: %+v", apiErr)
	}
	if attempts != 2 {
		t.Fatalf("unexpected retry attempts: %d", attempts)
	}
}
