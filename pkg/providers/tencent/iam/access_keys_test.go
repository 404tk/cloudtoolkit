package iam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListAccessKeysResolvesUinAndReturnsKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := readBody(t, r)
		switch r.Header.Get("X-TC-Action") {
		case "GetUser":
			if !strings.Contains(body, `"Name":"alice"`) {
				t.Fatalf("unexpected GetUser body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"Uin":42,"Name":"alice","RequestId":"r1"}}`))
		case "ListAccessKeys":
			if !strings.Contains(body, `"TargetUin":42`) {
				t.Fatalf("unexpected ListAccessKeys body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"AccessKeys":[{"AccessKeyId":"AKID111","Status":"Active","CreateTime":"2026-04-20 09:00:00"},{"AccessKeyId":"AKID222","Status":"Inactive","CreateTime":"2026-04-21 09:00:00"}],"RequestId":"r2"}}`))
		default:
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	keys, err := driver.ListAccessKeys(context.Background(), "alice")
	if err != nil {
		t.Fatalf("ListAccessKeys: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	if keys[0].CredentialID != "AKID111" || keys[0].CredentialType != "Active" {
		t.Errorf("unexpected first key: %+v", keys[0])
	}
	if keys[1].CredentialType != "Inactive" {
		t.Errorf("unexpected second key status: %+v", keys[1])
	}
}

func TestListAccessKeysAcceptsEmptyPrincipalForSelf(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-TC-Action"); got != "ListAccessKeys" {
			t.Fatalf("expected only ListAccessKeys when principal empty, got %s", got)
		}
		body := readBody(t, r)
		if strings.Contains(body, `TargetUin`) {
			t.Fatalf("expected no TargetUin for self, got: %s", body)
		}
		_, _ = w.Write([]byte(`{"Response":{"AccessKeys":[{"AccessKeyId":"AKIDSELF","Status":"Active","CreateTime":"2026-04-22 09:00:00"}],"RequestId":"r1"}}`))
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	keys, err := driver.ListAccessKeys(context.Background(), "")
	if err != nil {
		t.Fatalf("ListAccessKeys: %v", err)
	}
	if len(keys) != 1 || keys[0].CredentialID != "AKIDSELF" {
		t.Errorf("unexpected keys: %+v", keys)
	}
}

func TestCreateAccessKeyReturnsSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-TC-Action") {
		case "GetUser":
			_, _ = w.Write([]byte(`{"Response":{"Uin":99,"Name":"bob","RequestId":"r1"}}`))
		case "CreateAccessKey":
			body := readBody(t, r)
			if !strings.Contains(body, `"TargetUin":99`) {
				t.Fatalf("unexpected CreateAccessKey body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"AccessKey":{"AccessKeyId":"AKIDNEW","SecretAccessKey":"sekret","Status":"Active","CreateTime":"2026-04-30 09:00:00"},"RequestId":"r2"}}`))
		default:
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	cred, secret, err := driver.CreateAccessKey(context.Background(), "bob")
	if err != nil {
		t.Fatalf("CreateAccessKey: %v", err)
	}
	if cred.CredentialID != "AKIDNEW" {
		t.Errorf("unexpected key: %+v", cred)
	}
	if secret != "sekret" {
		t.Errorf("unexpected secret: %s", secret)
	}
}

func TestCreateAccessKeyRejectsEmptyPrincipal(t *testing.T) {
	driver := newTestDriver("http://example.invalid")
	if _, _, err := driver.CreateAccessKey(context.Background(), "  "); err == nil {
		t.Fatalf("expected error for empty principal")
	}
}

func TestDeleteAccessKeyPropagatesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-TC-Action") {
		case "GetUser":
			_, _ = w.Write([]byte(`{"Response":{"Uin":99,"Name":"bob","RequestId":"r1"}}`))
		case "DeleteAccessKey":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"Response":{"Error":{"Code":"ResourceNotFound.AccessKey","Message":"AK not found"},"RequestId":"r1"}}`))
		default:
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	err := driver.DeleteAccessKey(context.Background(), "bob", "AKIDINVALID")
	if err == nil {
		t.Fatalf("expected error from DeleteAccessKey")
	}
	if !strings.Contains(err.Error(), "ResourceNotFound") {
		t.Errorf("expected ResourceNotFound in error, got %v", err)
	}
}

func TestDeleteAccessKeyRejectsEmptyID(t *testing.T) {
	driver := newTestDriver("http://example.invalid")
	if err := driver.DeleteAccessKey(context.Background(), "alice", "  "); err == nil {
		t.Fatalf("expected error for empty access key id")
	}
}
