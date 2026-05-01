package iam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListRoleBindingsResolvesUinAndPaginates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := readBody(t, r)
		switch r.Header.Get("X-TC-Action") {
		case "GetUser":
			if !strings.Contains(body, `"Name":"alice"`) {
				t.Fatalf("unexpected GetUser body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"Uin":42,"Name":"alice","RequestId":"r1"}}`))
		case "ListAttachedUserAllPolicies":
			if !strings.Contains(body, `"TargetUin":42`) {
				t.Fatalf("unexpected ListAttachedUserAllPolicies body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"PolicyList":[{"PolicyId":"1","PolicyName":"AdministratorAccess","StrategyType":"2"}],"TotalNum":1,"RequestId":"r2"}}`))
		default:
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	got, err := driver.ListRoleBindings(context.Background(), "alice")
	if err != nil {
		t.Fatalf("ListRoleBindings: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 binding, got %d", len(got))
	}
	if got[0].Role != "AdministratorAccess" || got[0].Scope != "1" {
		t.Errorf("unexpected binding: %+v", got[0])
	}
	if got[0].Principal != "alice" {
		t.Errorf("expected principal 'alice', got %q", got[0].Principal)
	}
}

func TestListRoleBindingsRejectsEmptyPrincipal(t *testing.T) {
	driver := newTestDriver("http://example.invalid")
	if _, err := driver.ListRoleBindings(context.Background(), "  "); err == nil {
		t.Fatalf("expected error for empty principal")
	}
}

func TestAttachPolicyResolvesUinAndCallsAttachUserPolicy(t *testing.T) {
	var attached bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-TC-Action") {
		case "GetUser":
			_, _ = w.Write([]byte(`{"Response":{"Uin":99,"Name":"alice","RequestId":"r1"}}`))
		case "AttachUserPolicy":
			body := readBody(t, r)
			if !strings.Contains(body, `"AttachUin":99`) || !strings.Contains(body, `"PolicyId":1`) {
				t.Fatalf("unexpected AttachUserPolicy body: %s", body)
			}
			attached = true
			_, _ = w.Write([]byte(`{"Response":{"RequestId":"r2"}}`))
		default:
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	if err := driver.AttachPolicy(context.Background(), "alice", 1); err != nil {
		t.Fatalf("AttachPolicy: %v", err)
	}
	if !attached {
		t.Fatalf("AttachUserPolicy was not called")
	}
}

func TestDetachPolicyPropagatesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-TC-Action") {
		case "GetUser":
			_, _ = w.Write([]byte(`{"Response":{"Uin":99,"Name":"alice","RequestId":"r1"}}`))
		case "DetachUserPolicy":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"Response":{"Error":{"Code":"ResourceNotFound.PolicyIdNotFound","Message":"policy not attached"},"RequestId":"r2"}}`))
		default:
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	err := driver.DetachPolicy(context.Background(), "alice", 999)
	if err == nil {
		t.Fatalf("expected error from DetachPolicy")
	}
	if !strings.Contains(err.Error(), "ResourceNotFound.PolicyIdNotFound") {
		t.Errorf("expected ResourceNotFound.PolicyIdNotFound in error, got %v", err)
	}
}

func TestResolvePolicyID(t *testing.T) {
	cases := []struct {
		input string
		want  uint64
		err   bool
	}{
		{"1", 1, false},
		{"200001", 200001, false},
		{"AdministratorAccess", 1, false},
		{"administratoraccess", 1, false},
		{"QcloudResourceFullAccess", 1, false},
		{"", 0, true},
		{"NotARealPolicy", 0, true},
	}
	for _, c := range cases {
		got, err := ResolvePolicyID(c.input)
		if c.err {
			if err == nil {
				t.Errorf("ResolvePolicyID(%q) expected error", c.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ResolvePolicyID(%q) error: %v", c.input, err)
			continue
		}
		if got != c.want {
			t.Errorf("ResolvePolicyID(%q) = %d, want %d", c.input, got, c.want)
		}
	}
}
