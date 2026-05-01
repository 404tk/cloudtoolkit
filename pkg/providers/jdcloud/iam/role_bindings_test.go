package iam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListRoleBindingsParsesPolicies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/subUser/alice:describeAttachedPolicies" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"requestId":"r1","result":{"policies":[{"policyName":"JDCloudAdmin-New","policyType":"system"},{"policyName":"ReadOnly","policyType":"custom"}]}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL)}
	got, err := driver.ListRoleBindings(context.Background(), "alice")
	if err != nil {
		t.Fatalf("ListRoleBindings: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 bindings, got %d", len(got))
	}
	if got[0].Role != "JDCloudAdmin-New" || got[0].Scope != "system" {
		t.Errorf("unexpected first binding: %+v", got[0])
	}
	if got[0].Principal != "alice" {
		t.Errorf("expected principal 'alice', got %q", got[0].Principal)
	}
}

func TestListRoleBindingsRejectsEmptyPrincipal(t *testing.T) {
	driver := &Driver{Client: newTestClient("http://example.invalid")}
	if _, err := driver.ListRoleBindings(context.Background(), "  "); err == nil {
		t.Fatalf("expected error for empty principal")
	}
}

func TestAttachPolicyPostsExpectedBody(t *testing.T) {
	var captured string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/subUser/alice:attachSubUserPolicy" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		captured = readBody(t, r)
		_, _ = w.Write([]byte(`{"requestId":"r1","result":{}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL)}
	if err := driver.AttachPolicy(context.Background(), "alice", "JDCloudAdmin-New"); err != nil {
		t.Fatalf("AttachPolicy: %v", err)
	}
	if !strings.Contains(captured, `"subUser":"alice"`) || !strings.Contains(captured, `"policyName":"JDCloudAdmin-New"`) {
		t.Errorf("unexpected body: %s", captured)
	}
}

func TestDetachPolicyUsesDeleteAndQueryParam(t *testing.T) {
	var sawDelete bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/v1/subUser/alice:detachSubUserPolicy" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("policyName"); got != "JDCloudAdmin-New" {
			t.Fatalf("expected policyName=JDCloudAdmin-New in query, got %q", got)
		}
		sawDelete = true
		_, _ = w.Write([]byte(`{"requestId":"r1","result":{}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL)}
	if err := driver.DetachPolicy(context.Background(), "alice", "JDCloudAdmin-New"); err != nil {
		t.Fatalf("DetachPolicy: %v", err)
	}
	if !sawDelete {
		t.Fatalf("DELETE :detachSubUserPolicy was not called")
	}
}

func readBody(t *testing.T, r *http.Request) string {
	t.Helper()
	defer r.Body.Close()
	buf := make([]byte, 1024)
	n, _ := r.Body.Read(buf)
	return string(buf[:n])
}
