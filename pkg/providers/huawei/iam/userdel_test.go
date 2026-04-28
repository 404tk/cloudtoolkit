package iam

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDriverDelUserDeletesMatchedUser(t *testing.T) {
	stdout, stderr := withLoggerBuffers(t)
	var deletedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v5/users":
			_, _ = w.Write([]byte(`{"users":[{"user_id":"u-1","user_name":"ctk","enabled":true}]}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/v3/users/u-1":
			if got := r.Header.Get("X-Domain-Id"); got != "d-1" {
				t.Fatalf("unexpected delete-user domain header: %s", got)
			}
			deletedPath = r.URL.Path
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/v3/auth/domains":
			_, _ = w.Write([]byte(`{"domains":[{"id":"d-1","name":"example"}]}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "cn-north-4")
	driver.Username = "ctk"
	driver.DomainID = "d-1"
	result, err := driver.DelUser()

	if err != nil {
		t.Fatalf("DelUser failed: %v", err)
	}
	if deletedPath != "/v3/users/u-1" {
		t.Fatalf("unexpected delete path: %s", deletedPath)
	}
	if result.Username != "ctk" {
		t.Fatalf("unexpected username: %s", result.Username)
	}
	if !strings.Contains(result.Message, "success") {
		t.Fatalf("unexpected message: %s", result.Message)
	}
	// ListUsers is called internally and produces log output
	if !strings.Contains(stdout.String(), "List IAM users") {
		t.Fatalf("expected list users log in stdout: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr logs: %s", stderr.String())
	}
}

func TestDriverDelUserLogsNotFound(t *testing.T) {
	stdout, _ := withLoggerBuffers(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v5/users" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"users":[{"user_id":"u-1","user_name":"other","enabled":true}]}`))
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "cn-north-4")
	driver.Username = "ctk"
	result, err := driver.DelUser()

	if err == nil {
		t.Fatalf("expected error for user not found")
	}
	if !strings.Contains(err.Error(), "user ctk not found") {
		t.Fatalf("unexpected error message: %v", err)
	}
	// ListUsers is called internally and produces log output
	if !strings.Contains(stdout.String(), "List IAM users") {
		t.Fatalf("expected list users log in stdout: %s", stdout.String())
	}
	if result.Username != "" {
		t.Fatalf("expected empty username in error result, got: %s", result.Username)
	}
}
