package iam

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
)

func newRoleBindingDriver(baseURL string) *Driver {
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

func TestListRoleBindingsReturnsAttachedPolicies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("Action"); got != "ListPoliciesForUser" {
			t.Fatalf("unexpected action: %s", got)
		}
		if got := r.URL.Query().Get("UserName"); got != "demo" {
			t.Fatalf("unexpected user: %s", got)
		}
		_, _ = w.Write([]byte(`{"RequestId":"r1","Policies":{"Policy":[{"PolicyName":"AdministratorAccess","PolicyType":"System"},{"PolicyName":"ProjectReader","PolicyType":"Custom"}]}}`))
	}))
	defer server.Close()

	driver := newRoleBindingDriver(server.URL)
	bindings, err := driver.ListRoleBindings(context.Background(), "demo")
	if err != nil {
		t.Fatalf("ListRoleBindings: %v", err)
	}
	if len(bindings) != 2 {
		t.Fatalf("expected 2 bindings, got %d", len(bindings))
	}
	if bindings[0].Role != "AdministratorAccess" || bindings[0].Scope != "System" {
		t.Errorf("unexpected first binding: %+v", bindings[0])
	}
	if bindings[1].Role != "ProjectReader" || bindings[1].Scope != "Custom" {
		t.Errorf("unexpected second binding: %+v", bindings[1])
	}
	if bindings[0].Principal != "demo" {
		t.Errorf("expected principal 'demo', got %q", bindings[0].Principal)
	}
}

func TestListRoleBindingsRejectsEmptyPrincipal(t *testing.T) {
	driver := newRoleBindingDriver("http://example.invalid")
	if _, err := driver.ListRoleBindings(context.Background(), "  "); err == nil {
		t.Fatalf("expected error for empty principal")
	}
}

func TestAttachPolicyToUserSendsExpectedQuery(t *testing.T) {
	var captured map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = map[string]string{
			"Action":     r.URL.Query().Get("Action"),
			"UserName":   r.URL.Query().Get("UserName"),
			"PolicyName": r.URL.Query().Get("PolicyName"),
			"PolicyType": r.URL.Query().Get("PolicyType"),
		}
		_, _ = w.Write([]byte(`{"RequestId":"r1"}`))
	}))
	defer server.Close()

	driver := newRoleBindingDriver(server.URL)
	if err := driver.AttachPolicyToUser(context.Background(), "demo", "AliyunECSReadOnlyAccess", ""); err != nil {
		t.Fatalf("AttachPolicyToUser: %v", err)
	}
	if captured["Action"] != "AttachPolicyToUser" {
		t.Errorf("unexpected action: %s", captured["Action"])
	}
	if captured["UserName"] != "demo" {
		t.Errorf("unexpected user: %s", captured["UserName"])
	}
	if captured["PolicyName"] != "AliyunECSReadOnlyAccess" {
		t.Errorf("unexpected policy: %s", captured["PolicyName"])
	}
	if captured["PolicyType"] != "System" {
		t.Errorf("expected default System policy type, got %q", captured["PolicyType"])
	}
}

func TestDetachPolicyFromUserPropagatesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		body, _ := json.Marshal(map[string]string{
			"Code":      "EntityNotExist.User.Policy",
			"Message":   "The user policy does not exist.",
			"RequestId": "r1",
		})
		_, _ = w.Write(body)
	}))
	defer server.Close()

	driver := newRoleBindingDriver(server.URL)
	err := driver.DetachPolicyFromUser(context.Background(), "demo", "AliyunECSReadOnlyAccess", "Custom")
	if err == nil {
		t.Fatalf("expected error from DetachPolicyFromUser")
	}
	if !strings.Contains(err.Error(), "EntityNotExist.User.Policy") {
		t.Errorf("expected EntityNotExist.User.Policy in error, got %v", err)
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
