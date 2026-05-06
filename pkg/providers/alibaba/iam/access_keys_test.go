package iam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
)

func newAccessKeysDriver(baseURL string) *Driver {
	return &Driver{
		Cred:   aliauth.New("ak", "sk", ""),
		Region: "cn-hangzhou",
		clientOptions: []api.Option{
			api.WithBaseURL(baseURL),
			api.WithClock(func() time.Time { return time.Unix(1713376800, 0).UTC() }),
			api.WithNonce(func() string { return "nonce" }),
			api.WithRetryPolicy(api.RetryPolicy{
				MaxAttempts: 1,
				Sleep:       func(context.Context, time.Duration) error { return nil },
			}),
		},
	}
}

func TestListAccessKeysReturnsKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("Action"); got != "ListAccessKeys" {
			t.Fatalf("unexpected action: %s", got)
		}
		if got := r.URL.Query().Get("UserName"); got != "demo" {
			t.Fatalf("unexpected user: %s", got)
		}
		_, _ = w.Write([]byte(`{"RequestId":"r1","AccessKeys":{"AccessKey":[{"AccessKeyId":"LTAI4t1","Status":"Active","CreateDate":"2026-04-01T00:00:00Z"},{"AccessKeyId":"LTAI4t2","Status":"Inactive","CreateDate":"2026-04-02T00:00:00Z"}]}}`))
	}))
	defer server.Close()

	driver := newAccessKeysDriver(server.URL)
	keys, err := driver.ListAccessKeys(context.Background(), "demo")
	if err != nil {
		t.Fatalf("ListAccessKeys: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	if keys[0].CredentialID != "LTAI4t1" || keys[0].CredentialType != "Active" {
		t.Errorf("unexpected first key: %+v", keys[0])
	}
	if keys[1].CredentialType != "Inactive" {
		t.Errorf("unexpected second key status: %+v", keys[1])
	}
}

func TestCreateAccessKeyReturnsSecret(t *testing.T) {
	var captured string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.URL.Query().Get("UserName")
		if got := r.URL.Query().Get("Action"); got != "CreateAccessKey" {
			t.Fatalf("unexpected action: %s", got)
		}
		_, _ = w.Write([]byte(`{"RequestId":"r1","AccessKey":{"AccessKeyId":"LTAINEW","AccessKeySecret":"EXAMPLEsecret","Status":"Active","CreateDate":"2026-04-30T00:00:00Z"}}`))
	}))
	defer server.Close()

	driver := newAccessKeysDriver(server.URL)
	cred, secret, err := driver.CreateAccessKey(context.Background(), "demo")
	if err != nil {
		t.Fatalf("CreateAccessKey: %v", err)
	}
	if captured != "demo" {
		t.Errorf("unexpected captured user: %s", captured)
	}
	if cred.CredentialID != "LTAINEW" {
		t.Errorf("unexpected key: %+v", cred)
	}
	if secret != "EXAMPLEsecret" {
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
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"RequestId":"r1","Code":"EntityNotExist.User.AccessKey","Message":"The specified AccessKey does not exist."}`))
	}))
	defer server.Close()

	driver := newAccessKeysDriver(server.URL)
	err := driver.DeleteAccessKey(context.Background(), "demo", "LTAIINVALID")
	if err == nil {
		t.Fatalf("expected error from DeleteAccessKey")
	}
	if !strings.Contains(err.Error(), "EntityNotExist") {
		t.Errorf("expected EntityNotExist in error, got %v", err)
	}
}

func TestDeleteAccessKeyRejectsEmptyID(t *testing.T) {
	driver := newAccessKeysDriver("http://example.invalid")
	if err := driver.DeleteAccessKey(context.Background(), "demo", "  "); err == nil {
		t.Fatalf("expected error for empty access key id")
	}
}
