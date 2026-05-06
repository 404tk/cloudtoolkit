package iam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/auth"
)

func newAccessKeysDriver(baseURL string) *Driver {
	credential := auth.New("ucloudpubkey-EXAMPLE", "ucloudprivkey-EXAMPLE", "")
	return &Driver{
		Credential: credential,
		Client: api.NewClient(credential,
			api.WithBaseURL(baseURL),
			api.WithRetryPolicy(api.RetryPolicy{MaxAttempts: 1}),
		),
	}
}

func TestListAccessKeysParses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if got := r.Form.Get("Action"); got != "ListUserApiKeys" {
			t.Fatalf("unexpected action: %s", got)
		}
		if got := r.Form.Get("UserName"); got != "alice" {
			t.Fatalf("unexpected user: %s", got)
		}
		_, _ = w.Write([]byte(`{"Action":"ListUserApiKeysResponse","RetCode":0,"TotalCount":2,"ApiKeys":[{"AccessKeyID":"UC1","Status":"active","CreatedAt":"2026-04-20T08:00:00Z"},{"AccessKeyID":"UC2","Status":"inactive","CreatedAt":"2026-04-21T08:00:00Z"}]}`))
	}))
	defer server.Close()

	driver := newAccessKeysDriver(server.URL)
	keys, err := driver.ListAccessKeys(context.Background(), "alice")
	if err != nil {
		t.Fatalf("ListAccessKeys: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	if keys[0].CredentialID != "UC1" || keys[0].CredentialType != "active" {
		t.Errorf("unexpected first key: %+v", keys[0])
	}
}

func TestCreateAccessKeyReturnsSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if got := r.Form.Get("Action"); got != "CreateUserApiKey" {
			t.Fatalf("unexpected action: %s", got)
		}
		_, _ = w.Write([]byte(`{"Action":"CreateUserApiKeyResponse","RetCode":0,"AccessKeyID":"UCNEW","AccessKeySecret":"sekret","Status":"active","CreatedAt":"2026-04-30T09:00:00Z"}`))
	}))
	defer server.Close()

	driver := newAccessKeysDriver(server.URL)
	cred, secret, err := driver.CreateAccessKey(context.Background(), "alice")
	if err != nil {
		t.Fatalf("CreateAccessKey: %v", err)
	}
	if cred.CredentialID != "UCNEW" {
		t.Errorf("unexpected key: %+v", cred)
	}
	if secret != "sekret" {
		t.Errorf("unexpected secret: %s", secret)
	}
}

func TestCreateAccessKeyRejectsEmptyPrincipal(t *testing.T) {
	driver := newAccessKeysDriver("http://example.invalid")
	if _, _, err := driver.CreateAccessKey(context.Background(), "  "); err == nil {
		t.Fatalf("expected error for empty principal")
	}
}

func TestDeleteAccessKeyPropagatesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Action":"DeleteUserApiKeyResponse","RetCode":1011,"Message":"AK not found"}`))
	}))
	defer server.Close()

	driver := newAccessKeysDriver(server.URL)
	err := driver.DeleteAccessKey(context.Background(), "alice", "UCINVALID")
	if err == nil {
		t.Fatalf("expected error from DeleteAccessKey")
	}
	if !strings.Contains(err.Error(), "AK not found") {
		t.Errorf("expected error message to contain not found, got %v", err)
	}
}

func TestDeleteAccessKeyRejectsEmpty(t *testing.T) {
	driver := newAccessKeysDriver("http://example.invalid")
	if err := driver.DeleteAccessKey(context.Background(), "", "UCID"); err == nil {
		t.Fatalf("expected error for empty user")
	}
	if err := driver.DeleteAccessKey(context.Background(), "alice", "  "); err == nil {
		t.Fatalf("expected error for empty access key id")
	}
}
