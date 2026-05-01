package iam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListRoleBindingsResolvesUserAndReturnsGroups(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v5/users":
			_, _ = w.Write([]byte(`{"users":[{"user_id":"u-99","user_name":"alice","enabled":true}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v3/auth/domains":
			_, _ = w.Write([]byte(`{"domains":[{"id":"dom-1","name":"ctk-demo"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v3/users/u-99/groups":
			_, _ = w.Write([]byte(`{"groups":[{"id":"g-1","name":"admin"},{"id":"g-2","name":"readonly"}]}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "cn-north-4")
	got, err := driver.ListRoleBindings(context.Background(), "alice")
	if err != nil {
		t.Fatalf("ListRoleBindings: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 bindings, got %d", len(got))
	}
	if got[0].Role != "admin" || got[0].AssignmentID != "g-1" {
		t.Errorf("unexpected first binding: %+v", got[0])
	}
	if got[1].Role != "readonly" || got[1].AssignmentID != "g-2" {
		t.Errorf("unexpected second binding: %+v", got[1])
	}
}

func TestListRoleBindingsRejectsEmptyPrincipal(t *testing.T) {
	driver := newTestDriver("http://example.invalid", "cn-north-4")
	if _, err := driver.ListRoleBindings(context.Background(), "  "); err == nil {
		t.Fatalf("expected error for empty principal")
	}
}

func TestListRoleBindingsErrorsWhenUserMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v5/users" {
			_, _ = w.Write([]byte(`{"users":[{"user_id":"u-1","user_name":"someoneelse","enabled":true}]}`))
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "cn-north-4")
	_, err := driver.ListRoleBindings(context.Background(), "alice")
	if err == nil {
		t.Fatalf("expected error for missing user")
	}
	if !strings.Contains(err.Error(), `user "alice" not found`) {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAttachGroupResolvesIDsAndPUTs(t *testing.T) {
	var sawPut bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v5/users":
			_, _ = w.Write([]byte(`{"users":[{"user_id":"u-99","user_name":"alice","enabled":true}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v3/auth/domains":
			_, _ = w.Write([]byte(`{"domains":[{"id":"dom-1","name":"ctk-demo"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v3/groups":
			_, _ = w.Write([]byte(`{"groups":[{"id":"g-7","name":"admin"}]}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v3/groups/g-7/users/u-99":
			sawPut = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "cn-north-4")
	if err := driver.AttachGroup(context.Background(), "alice", "admin"); err != nil {
		t.Fatalf("AttachGroup: %v", err)
	}
	if !sawPut {
		t.Fatalf("expected PUT /v3/groups/g-7/users/u-99")
	}
}

func TestDetachGroupSendsDelete(t *testing.T) {
	var sawDelete bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v5/users":
			_, _ = w.Write([]byte(`{"users":[{"user_id":"u-99","user_name":"alice","enabled":true}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v3/auth/domains":
			_, _ = w.Write([]byte(`{"domains":[{"id":"dom-1","name":"ctk-demo"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v3/groups":
			_, _ = w.Write([]byte(`{"groups":[{"id":"g-7","name":"admin"}]}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/v3/groups/g-7/users/u-99":
			sawDelete = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "cn-north-4")
	if err := driver.DetachGroup(context.Background(), "alice", "admin"); err != nil {
		t.Fatalf("DetachGroup: %v", err)
	}
	if !sawDelete {
		t.Fatalf("expected DELETE /v3/groups/g-7/users/u-99")
	}
}
