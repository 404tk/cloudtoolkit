package api

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSignRejectsEmptyNonce(t *testing.T) {
	_, err := Sign(SignInput{
		Method:      http.MethodGet,
		Host:        "iam.jdcloud-api.com",
		Path:        "/v1/subUsers",
		ContentType: "application/json",
		Service:     "iam",
		Region:      "jdcloud-api",
		AccessKey:   "AKID",
		SecretKey:   "SECRET",
		Timestamp:   time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC),
	})
	if err == nil || !strings.Contains(err.Error(), "empty nonce") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSignIgnoresUserAgentButSignsCustomHeaders(t *testing.T) {
	got, err := Sign(SignInput{
		Method:      http.MethodGet,
		Host:        "iam.jdcloud-api.com",
		Path:        "/v1/subUsers",
		ContentType: "application/json",
		Service:     "iam",
		Region:      "jdcloud-api",
		AccessKey:   "AKID",
		SecretKey:   "SECRET",
		Nonce:       "ebf8b26d-c3be-402f-9f10-f8b6573fd823",
		Timestamp:   time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC),
		Headers: http.Header{
			"User-Agent":           []string{"ctk-test"},
			"X-Jdcloud-Request-Id": []string{"req-local"},
			"X-Custom-Debug":       []string{"trace"},
		},
	})
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	if strings.Contains(got.SignedHeaders, "user-agent") || strings.Contains(got.SignedHeaders, "x-jdcloud-request-id") {
		t.Fatalf("ignored headers should not be signed: %s", got.SignedHeaders)
	}
	if !strings.Contains(got.SignedHeaders, "x-custom-debug") {
		t.Fatalf("custom header should be signed: %s", got.SignedHeaders)
	}
	if !strings.Contains(got.CanonicalRequest, "x-custom-debug:trace") {
		t.Fatalf("canonical request missing custom header: %s", got.CanonicalRequest)
	}
}
