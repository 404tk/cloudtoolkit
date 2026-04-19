package iam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDriverListUsersMapsResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v5/users" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"users":[{"user_id":"u-1","user_name":"alice","enabled":true},{"user_id":"u-2","user_name":"bob","enabled":false}]}`))
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "cn-north-4")
	got, err := driver.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("unexpected user count: %d", len(got))
	}
	if got[0].UserName != "alice" || got[0].UserId != "u-1" || !got[0].EnableLogin {
		t.Fatalf("unexpected first user: %+v", got[0])
	}
	if got[1].UserName != "bob" || got[1].UserId != "u-2" || got[1].EnableLogin {
		t.Fatalf("unexpected second user: %+v", got[1])
	}
}

func TestDriverListUsersRejectsAllRegion(t *testing.T) {
	driver := newTestDriver("https://example.com", "all")
	_, err := driver.ListUsers(context.Background())
	if err == nil {
		t.Fatal("expected unresolved region error")
	}
}

func TestDriverListUsersDoesNotProbeDomainsWhenDomainIDMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v5/users":
			_, _ = w.Write([]byte(`{"users":[{"user_id":"u-1","user_name":"alice","enabled":true}]}`))
		case "/v3/auth/domains":
			t.Fatal("ListUsers() should not probe /v3/auth/domains when DomainID is missing")
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "cn-north-4")
	got, err := driver.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(got) != 1 || got[0].UserName != "alice" {
		t.Fatalf("unexpected users: %+v", got)
	}
}
