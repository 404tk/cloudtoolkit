package iam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListAccessKeysResolvesUserIDAndReturnsKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v5/users":
			_, _ = w.Write([]byte(`{"users":[{"user_id":"U-42","user_name":"alice","enabled":true}]}`))
		case "/v3.0/OS-CREDENTIAL/credentials":
			if got := r.URL.Query().Get("user_id"); got != "U-42" {
				t.Fatalf("unexpected user_id: %s", got)
			}
			_, _ = w.Write([]byte(`{"credentials":[{"access":"AKHW1","user_id":"U-42","status":"active","create_time":"2026-04-20T08:00:00.000000"},{"access":"AKHW2","user_id":"U-42","status":"inactive","create_time":"2026-04-21T08:00:00.000000"}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "cn-north-4")
	keys, err := driver.ListAccessKeys(context.Background(), "alice")
	if err != nil {
		t.Fatalf("ListAccessKeys: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	if keys[0].CredentialID != "AKHW1" || keys[0].CredentialType != "active" {
		t.Errorf("unexpected first key: %+v", keys[0])
	}
	if keys[1].CredentialType != "inactive" {
		t.Errorf("unexpected second key status: %+v", keys[1])
	}
}

func TestCreateAccessKeyReturnsSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v5/users":
			_, _ = w.Write([]byte(`{"users":[{"user_id":"U-99","user_name":"bob","enabled":true}]}`))
		case "/v3.0/OS-CREDENTIAL/credentials":
			body := readBody(t, r)
			if !strings.Contains(body, `"user_id":"U-99"`) {
				t.Fatalf("unexpected create body: %s", body)
			}
			_, _ = w.Write([]byte(`{"credential":{"access":"AKNEW","secret":"sekret","user_id":"U-99","status":"active","create_time":"2026-04-30T09:00:00.000000"}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "cn-north-4")
	cred, secret, err := driver.CreateAccessKey(context.Background(), "bob")
	if err != nil {
		t.Fatalf("CreateAccessKey: %v", err)
	}
	if cred.CredentialID != "AKNEW" {
		t.Errorf("unexpected key: %+v", cred)
	}
	if secret != "sekret" {
		t.Errorf("unexpected secret: %s", secret)
	}
}

func TestCreateAccessKeyRejectsEmptyPrincipal(t *testing.T) {
	driver := newTestDriver("http://example.invalid", "cn-north-4")
	if _, _, err := driver.CreateAccessKey(context.Background(), "  "); err == nil {
		t.Fatalf("expected error for empty principal")
	}
}

func TestDeleteAccessKeyPropagatesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"IAM.0007","message":"AK not found"}}`))
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "cn-north-4")
	err := driver.DeleteAccessKey(context.Background(), "", "AKINVALID")
	if err == nil {
		t.Fatalf("expected error from DeleteAccessKey")
	}
	if !strings.Contains(err.Error(), "IAM.0007") {
		t.Errorf("expected IAM.0007 in error, got %v", err)
	}
}

func TestDeleteAccessKeyRejectsEmptyID(t *testing.T) {
	driver := newTestDriver("http://example.invalid", "cn-north-4")
	if err := driver.DeleteAccessKey(context.Background(), "", "  "); err == nil {
		t.Fatalf("expected error for empty access key id")
	}
}
