package iam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDriverGetUserName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v3.0/OS-CREDENTIAL/credentials/AKID":
			_, _ = w.Write([]byte(`{"credential":{"user_id":"u-123"}}`))
		case "/v3/users/u-123":
			_, _ = w.Write([]byte(`{"user":{"id":"u-123","name":"alice","domain_id":"d-1"}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "cn-north-4")
	got, err := driver.GetUserName(context.Background())
	if err != nil {
		t.Fatalf("GetUserName() error = %v", err)
	}
	if got != "alice" {
		t.Fatalf("GetUserName() = %q, want %q", got, "alice")
	}
	if driver.DomainID != "d-1" {
		t.Fatalf("unexpected cached domain id: %q", driver.DomainID)
	}
}

func TestDriverGetUserIDReturnsLegacyErrorMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"credential":{"user_id":""},"error_msg":"invalid ak"}`))
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "cn-north-4")
	_, err := driver.getUserID(context.Background())
	if err == nil || err.Error() != "invalid ak" {
		t.Fatalf("unexpected error: %v", err)
	}
}
