package httpclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestExecuteRetriesWithReusableBody(t *testing.T) {
	var attempts int
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		body, err := SnapshotBody(&http.Response{Body: r.Body})
		if err != nil {
			t.Fatalf("snapshot request body: %v", err)
		}
		bodies = append(bodies, string(body))
		if attempts == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`retry`))
			return
		}
		_, _ = w.Write([]byte(`ok`))
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(`{"name":"demo"}`))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, body, err := Execute(context.Background(), server.Client(), DefaultRetryPolicy(), req, true)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if attempts != 2 {
		t.Fatalf("unexpected attempts: %d", attempts)
	}
	if len(bodies) != 2 || bodies[0] != `{"name":"demo"}` || bodies[1] != `{"name":"demo"}` {
		t.Fatalf("unexpected request bodies: %v", bodies)
	}
	if string(body) != "ok" {
		t.Fatalf("unexpected body: %s", string(body))
	}
	if resp == nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestRetryPolicyHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := DefaultRetryPolicy().Do(ctx, true, func() (*http.Response, error) {
		return nil, errors.New("network down")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func TestDecodeJSONNoContent(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusNoContent}
	var payload map[string]any
	if err := DecodeJSON(resp, nil, "demo", &payload); err != nil {
		t.Fatalf("DecodeJSON() error = %v", err)
	}
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)

	delay, ok := ParseRetryAfter("2", now)
	if !ok || delay != 2*time.Second {
		t.Fatalf("unexpected seconds retry-after: delay=%v ok=%v", delay, ok)
	}

	delay, ok = ParseRetryAfter("Mon, 27 Apr 2026 12:00:03 GMT", now)
	if !ok || delay != 3*time.Second {
		t.Fatalf("unexpected date retry-after: delay=%v ok=%v", delay, ok)
	}
}
