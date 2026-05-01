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

func newRoleBindingDriver(baseURL, projectID string) *Driver {
	credential := auth.New("ucloudpubkey-EXAMPLE", "ucloudprivkey-EXAMPLE", "")
	return &Driver{
		Credential: credential,
		Client: api.NewClient(credential,
			api.WithBaseURL(baseURL),
			api.WithRetryPolicy(api.RetryPolicy{MaxAttempts: 1}),
		),
		ProjectID: projectID,
	}
}

func TestListRoleBindingsParsesPolicies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("Action") != "ListPoliciesForUser" {
			t.Fatalf("unexpected action: %s", r.Form.Get("Action"))
		}
		if r.Form.Get("UserName") != "alice" {
			t.Fatalf("unexpected user: %s", r.Form.Get("UserName"))
		}
		_, _ = w.Write([]byte(`{"Action":"ListPoliciesForUserResponse","RetCode":0,"TotalCount":2,"Policies":[{"PolicyURN":"ucs:iam::ucs:policy/AdministratorAccess","PolicyName":"AdministratorAccess","Scope":"Specified","ProjectID":"org-demo"},{"PolicyURN":"ucs:iam::ucs:policy/IAMFullAccess","PolicyName":"IAMFullAccess","Scope":"Unspecified"}]}`))
	}))
	defer server.Close()

	driver := newRoleBindingDriver(server.URL, "org-demo")
	got, err := driver.ListRoleBindings(context.Background(), "alice")
	if err != nil {
		t.Fatalf("ListRoleBindings: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 bindings, got %d", len(got))
	}
	if got[0].Role != "AdministratorAccess" || got[0].Scope != "Specified" {
		t.Errorf("unexpected first binding: %+v", got[0])
	}
	if got[1].Scope != "Unspecified" {
		t.Errorf("unexpected second scope: %s", got[1].Scope)
	}
	if got[0].AssignmentID != "ucs:iam::ucs:policy/AdministratorAccess" {
		t.Errorf("expected URN as AssignmentID, got %q", got[0].AssignmentID)
	}
}

func TestListRoleBindingsRejectsEmptyPrincipal(t *testing.T) {
	driver := newRoleBindingDriver("http://example.invalid", "")
	if _, err := driver.ListRoleBindings(context.Background(), "  "); err == nil {
		t.Fatalf("expected error for empty principal")
	}
}

func TestAttachPolicyDefaultsScopeToUnspecified(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("Action") != "AttachPoliciesToUser" {
			t.Fatalf("unexpected action: %s", r.Form.Get("Action"))
		}
		if got := r.Form.Get("Scope"); got != "Unspecified" {
			t.Errorf("expected default Scope=Unspecified, got %q", got)
		}
		if r.Form.Get("PolicyURNs.0") != "ucs:iam::ucs:policy/AdministratorAccess" {
			t.Errorf("unexpected PolicyURN: %s", r.Form.Get("PolicyURNs.0"))
		}
		_, _ = w.Write([]byte(`{"Action":"AttachPoliciesToUserResponse","RetCode":0}`))
	}))
	defer server.Close()

	driver := newRoleBindingDriver(server.URL, "")
	if err := driver.AttachPolicy(context.Background(), "alice", "ucs:iam::ucs:policy/AdministratorAccess", ""); err != nil {
		t.Fatalf("AttachPolicy: %v", err)
	}
}

func TestAttachPolicyRequiresProjectIDWhenScoped(t *testing.T) {
	driver := newRoleBindingDriver("http://example.invalid", "")
	err := driver.AttachPolicy(context.Background(), "alice", "ucs:iam::ucs:policy/AdministratorAccess", "Specified")
	if err == nil || !strings.Contains(err.Error(), "ProjectID") {
		t.Fatalf("expected ProjectID required error, got %v", err)
	}
}

func TestDetachPolicySendsExpectedAction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("Action") != "DetachPoliciesFromUser" {
			t.Fatalf("unexpected action: %s", r.Form.Get("Action"))
		}
		_, _ = w.Write([]byte(`{"Action":"DetachPoliciesFromUserResponse","RetCode":0}`))
	}))
	defer server.Close()

	driver := newRoleBindingDriver(server.URL, "")
	if err := driver.DetachPolicy(context.Background(), "alice", "ucs:iam::ucs:policy/AdministratorAccess", ""); err != nil {
		t.Fatalf("DetachPolicy: %v", err)
	}
}

func TestResolvePolicyURN(t *testing.T) {
	cases := map[string]string{
		"":                                    "",
		"AdministratorAccess":                 "ucs:iam::ucs:policy/AdministratorAccess",
		"IAMFullAccess":                       "ucs:iam::ucs:policy/IAMFullAccess",
		"ucs:iam::ucs:policy/AdministratorAccess": "ucs:iam::ucs:policy/AdministratorAccess",
	}
	for in, want := range cases {
		if got := ResolvePolicyURN(in); got != want {
			t.Errorf("ResolvePolicyURN(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeUCloudScope(t *testing.T) {
	cases := map[string]string{
		"":            "Unspecified",
		"unspecified": "Unspecified",
		"global":      "Unspecified",
		"specified":   "Specified",
		"project":     "Specified",
		"Custom":      "Custom",
	}
	for in, want := range cases {
		if got := normalizeUCloudScope(in); got != want {
			t.Errorf("normalizeUCloudScope(%q) = %q, want %q", in, got, want)
		}
	}
}
