package iam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestListAccessKeysParsesMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values, _ := url.ParseQuery(r.URL.RawQuery)
		if values.Get("Action") != "ListAccessKeys" {
			t.Fatalf("unexpected action: %s", values.Get("Action"))
		}
		if values.Get("UserName") != "alice" {
			t.Fatalf("unexpected user: %s", values.Get("UserName"))
		}
		_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"r1"},"Result":{"AccessKeyMetadata":[{"AccessKeyId":"AKLT1","Status":"Active","CreateDate":"20260420T090000Z"},{"AccessKeyId":"AKLT2","Status":"Inactive","CreateDate":"20260421T090000Z"}]}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-beijing"}
	keys, err := driver.ListAccessKeys(context.Background(), "alice")
	if err != nil {
		t.Fatalf("ListAccessKeys: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	if keys[0].CredentialID != "AKLT1" || keys[0].CredentialType != "Active" {
		t.Errorf("unexpected first key: %+v", keys[0])
	}
	if keys[1].CredentialType != "Inactive" {
		t.Errorf("unexpected second key status: %+v", keys[1])
	}
}

func TestCreateAccessKeyReturnsSecret(t *testing.T) {
	var captured url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured, _ = url.ParseQuery(r.URL.RawQuery)
		_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"r1"},"Result":{"AccessKey":{"AccessKeyId":"AKLTNEW","SecretAccessKey":"sekret","Status":"Active","CreateDate":"20260430T090000Z"}}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-beijing"}
	cred, secret, err := driver.CreateAccessKey(context.Background(), "alice")
	if err != nil {
		t.Fatalf("CreateAccessKey: %v", err)
	}
	if captured.Get("Action") != "CreateAccessKey" {
		t.Errorf("unexpected action: %s", captured.Get("Action"))
	}
	if cred.CredentialID != "AKLTNEW" {
		t.Errorf("unexpected key: %+v", cred)
	}
	if secret != "sekret" {
		t.Errorf("unexpected secret: %s", secret)
	}
}

func TestCreateAccessKeyRejectsEmptyPrincipal(t *testing.T) {
	driver := &Driver{Client: newTestClient("http://example.invalid"), Region: "cn-beijing"}
	if _, _, err := driver.CreateAccessKey(context.Background(), "  "); err == nil {
		t.Fatalf("expected error for empty principal")
	}
}

func TestDeleteAccessKeyPropagatesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"r1","Error":{"Code":"ResourceNotFound.AccessKey","Message":"AK not found"}}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-beijing"}
	err := driver.DeleteAccessKey(context.Background(), "alice", "AKLTINVALID")
	if err == nil {
		t.Fatalf("expected error from DeleteAccessKey")
	}
	if !strings.Contains(err.Error(), "ResourceNotFound") {
		t.Errorf("expected ResourceNotFound in error, got %v", err)
	}
}

func TestDeleteAccessKeyRejectsEmpty(t *testing.T) {
	driver := &Driver{Client: newTestClient("http://example.invalid"), Region: "cn-beijing"}
	if err := driver.DeleteAccessKey(context.Background(), "", "AKLTID"); err == nil {
		t.Fatalf("expected error for empty user")
	}
	if err := driver.DeleteAccessKey(context.Background(), "alice", "  "); err == nil {
		t.Fatalf("expected error for empty access key id")
	}
}
