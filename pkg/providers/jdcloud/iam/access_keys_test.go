package iam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListAccessKeysParses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/subUser/alice:describeAccessKeys" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"requestId":"r1","result":{"accessKeys":[{"accessKey":"JDC_AK1","status":"active","createTime":"2026-04-20T08:00:00Z"},{"accessKey":"JDC_AK2","status":"inactive","createTime":"2026-04-21T08:00:00Z"}]}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL)}
	keys, err := driver.ListAccessKeys(context.Background(), "alice")
	if err != nil {
		t.Fatalf("ListAccessKeys: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	if keys[0].CredentialID != "JDC_AK1" || keys[0].CredentialType != "active" {
		t.Errorf("unexpected first key: %+v", keys[0])
	}
	if keys[1].CredentialType != "inactive" {
		t.Errorf("unexpected second key status: %+v", keys[1])
	}
}

func TestCreateAccessKeyReturnsSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/subUser/alice:createAccessKey" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"requestId":"r1","result":{"accessKey":{"accessKey":"JDC_AKNEW","secretKey":"sekret","status":"active","createTime":"2026-04-30T09:00:00Z"}}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL)}
	cred, secret, err := driver.CreateAccessKey(context.Background(), "alice")
	if err != nil {
		t.Fatalf("CreateAccessKey: %v", err)
	}
	if cred.CredentialID != "JDC_AKNEW" {
		t.Errorf("unexpected key: %+v", cred)
	}
	if secret != "sekret" {
		t.Errorf("unexpected secret: %s", secret)
	}
}

func TestCreateAccessKeyRejectsEmptyPrincipal(t *testing.T) {
	driver := &Driver{Client: newTestClient("http://example.invalid")}
	if _, _, err := driver.CreateAccessKey(context.Background(), "  "); err == nil {
		t.Fatalf("expected error for empty principal")
	}
}

func TestDeleteAccessKeyPropagatesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/v1/subUser/alice:deleteAccessKey" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("accessKey"); got != "JDC_INVALID" {
			t.Fatalf("unexpected access key: %s", got)
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"requestId":"r1","error":{"code":1011,"message":"AK not found"}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL)}
	err := driver.DeleteAccessKey(context.Background(), "alice", "JDC_INVALID")
	if err == nil {
		t.Fatalf("expected error from DeleteAccessKey")
	}
	if !strings.Contains(err.Error(), "AK not found") {
		t.Errorf("expected error message to contain not found, got %v", err)
	}
}

func TestDeleteAccessKeyRejectsEmptyID(t *testing.T) {
	driver := &Driver{Client: newTestClient("http://example.invalid")}
	if err := driver.DeleteAccessKey(context.Background(), "alice", "  "); err == nil {
		t.Fatalf("expected error for empty access key id")
	}
}
