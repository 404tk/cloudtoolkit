package iam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestListRoleBindingsParsesPolicies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values, _ := url.ParseQuery(r.URL.RawQuery)
		if values.Get("Action") != "ListAttachedUserPolicies" {
			t.Fatalf("unexpected action: %s", values.Get("Action"))
		}
		if values.Get("UserName") != "alice" {
			t.Fatalf("unexpected user: %s", values.Get("UserName"))
		}
		_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"r1"},"Result":{"AttachedPolicyMetadata":[{"PolicyName":"AdministratorAccess","PolicyType":"System","PolicyTrn":"trn:iam:::policy/AdministratorAccess"},{"PolicyName":"ReadOnly","PolicyType":"Custom","PolicyTrn":"trn:iam:10001:policy/ReadOnly"}]}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-beijing"}
	got, err := driver.ListRoleBindings(context.Background(), "alice")
	if err != nil {
		t.Fatalf("ListRoleBindings: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 bindings, got %d", len(got))
	}
	if got[0].Role != "AdministratorAccess" || got[0].Scope != "System" || got[0].AssignmentID != "trn:iam:::policy/AdministratorAccess" {
		t.Errorf("unexpected first binding: %+v", got[0])
	}
	if got[1].Role != "ReadOnly" || got[1].Scope != "Custom" {
		t.Errorf("unexpected second binding: %+v", got[1])
	}
	if got[0].Principal != "alice" {
		t.Errorf("expected principal 'alice', got %q", got[0].Principal)
	}
}

func TestListRoleBindingsRejectsEmptyPrincipal(t *testing.T) {
	driver := &Driver{Client: newTestClient("http://example.invalid"), Region: "cn-beijing"}
	if _, err := driver.ListRoleBindings(context.Background(), "  "); err == nil {
		t.Fatalf("expected error for empty principal")
	}
}

func TestAttachPolicyDefaultsPolicyTypeToSystem(t *testing.T) {
	var captured url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured, _ = url.ParseQuery(r.URL.RawQuery)
		_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"r1"}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-beijing"}
	if err := driver.AttachPolicy(context.Background(), "alice", "AdministratorAccess", ""); err != nil {
		t.Fatalf("AttachPolicy: %v", err)
	}
	if captured.Get("Action") != "AttachUserPolicy" {
		t.Errorf("unexpected action: %s", captured.Get("Action"))
	}
	if captured.Get("UserName") != "alice" || captured.Get("PolicyName") != "AdministratorAccess" {
		t.Errorf("unexpected query: %+v", captured)
	}
	if captured.Get("PolicyType") != "System" {
		t.Errorf("expected default System policy type, got %q", captured.Get("PolicyType"))
	}
}

func TestDetachPolicyPropagatesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"r1","Error":{"Code":"EntityNotExist.UserPolicy","Message":"policy not attached"}}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-beijing"}
	err := driver.DetachPolicy(context.Background(), "alice", "Never", "Custom")
	if err == nil {
		t.Fatalf("expected error from DetachPolicy")
	}
	if !strings.Contains(err.Error(), "EntityNotExist.UserPolicy") {
		t.Errorf("expected EntityNotExist.UserPolicy in error, got %v", err)
	}
}

func TestNormalizePolicyType(t *testing.T) {
	cases := map[string]string{
		"":         "System",
		" ":        "System",
		"system":   "System",
		"Custom":   "Custom",
		"custom":   "Custom",
		"Detached": "Detached",
	}
	for in, want := range cases {
		if got := normalizePolicyType(in); got != want {
			t.Errorf("normalizePolicyType(%q) = %q, want %q", in, got, want)
		}
	}
}
