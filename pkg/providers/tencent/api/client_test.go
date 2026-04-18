package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
)

func TestClientDoJSONBuildsTencentRequest(t *testing.T) {
	fixedTime := time.Unix(1704164645, 0).UTC()
	type requestSnapshot struct {
		Method        string
		Action        string
		Version       string
		Region        string
		RequestClient string
		Language      string
		Timestamp     string
		Authorization string
		Host          string
		Body          string
	}
	snapshots := make(chan requestSnapshot, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		snapshots <- requestSnapshot{
			Method:        r.Method,
			Action:        r.Header.Get("X-TC-Action"),
			Version:       r.Header.Get("X-TC-Version"),
			Region:        r.Header.Get("X-TC-Region"),
			RequestClient: r.Header.Get("X-TC-RequestClient"),
			Language:      r.Header.Get("X-TC-Language"),
			Timestamp:     r.Header.Get("X-TC-Timestamp"),
			Authorization: r.Header.Get("Authorization"),
			Host:          r.Host,
			Body:          string(body),
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Response":{"Arn":"qcs::cam::uin/10001:uin/10001","Type":"RootAccount","RequestId":"req-1"}}`))
	}))
	defer server.Close()

	client := NewClient(
		auth.New("AKIDEXAMPLE", "secretExampleKey", ""),
		WithBaseURL(server.URL),
		WithClock(func() time.Time { return fixedTime }),
		WithRequestClient("ctk-test"),
		WithLanguage("zh-CN"),
		WithRetryPolicy(RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)

	var resp GetCallerIdentityResponse
	if err := client.DoJSON(
		context.Background(),
		"sts",
		"2018-08-13",
		"GetCallerIdentity",
		"ap-guangzhou",
		GetCallerIdentityRequest{},
		&resp,
	); err != nil {
		t.Fatalf("DoJSON() error = %v", err)
	}
	if resp.Response.Arn != "qcs::cam::uin/10001:uin/10001" {
		t.Fatalf("unexpected arn: %s", resp.Response.Arn)
	}
	got := <-snapshots
	if got.Method != http.MethodPost {
		t.Fatalf("unexpected method: %s", got.Method)
	}
	if got.Action != "GetCallerIdentity" {
		t.Fatalf("unexpected action header: %s", got.Action)
	}
	if got.Version != "2018-08-13" {
		t.Fatalf("unexpected version header: %s", got.Version)
	}
	if got.Region != "ap-guangzhou" {
		t.Fatalf("unexpected region header: %s", got.Region)
	}
	if got.RequestClient != "ctk-test" {
		t.Fatalf("unexpected request client: %s", got.RequestClient)
	}
	if got.Language != "zh-CN" {
		t.Fatalf("unexpected language: %s", got.Language)
	}
	if got.Timestamp != "1704164645" {
		t.Fatalf("unexpected timestamp: %s", got.Timestamp)
	}
	if got.Authorization == "" {
		t.Fatal("missing authorization header")
	}
	if got.Host == "" {
		t.Fatal("missing host")
	}
	if got.Body != "{}" {
		t.Fatalf("unexpected body: %s", got.Body)
	}
}

func TestClientDoJSONReturnsTencentAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Response":{"Error":{"Code":"AuthFailure.SignatureFailure","Message":"signature mismatch"},"RequestId":"req-2"}}`))
	}))
	defer server.Close()

	client := NewClient(
		auth.New("AKIDEXAMPLE", "secretExampleKey", ""),
		WithBaseURL(server.URL),
		WithRetryPolicy(RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)

	err := client.DoJSON(
		context.Background(),
		"sts",
		"2018-08-13",
		"GetCallerIdentity",
		"ap-guangzhou",
		GetCallerIdentityRequest{},
		&GetCallerIdentityResponse{},
	)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %v", err)
	}
	if apiErr.Code != "AuthFailure.SignatureFailure" {
		t.Fatalf("unexpected error code: %s", apiErr.Code)
	}
}
