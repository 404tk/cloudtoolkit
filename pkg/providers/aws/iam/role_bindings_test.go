package iam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/aws/auth"
)

func newRoleBindingTestDriver(baseURL string) *Driver {
	return &Driver{
		Client: api.NewClient(
			auth.New("AKID", "SECRET", ""),
			api.WithBaseURL(baseURL),
			api.WithClock(func() time.Time { return time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC) }),
			api.WithRetryPolicy(api.RetryPolicy{
				MaxAttempts: 1,
				Sleep:       func(context.Context, time.Duration) error { return nil },
			}),
		),
		Region: "us-east-1",
	}
}

func parseRoleBindingForm(t *testing.T, r *http.Request) url.Values {
	t.Helper()
	if err := r.ParseForm(); err != nil {
		t.Fatalf("ParseForm: %v", err)
	}
	return r.PostForm
}

func TestListRoleBindingsParsesPolicies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values := parseRoleBindingForm(t, r)
		if got := values.Get("Action"); got != "ListAttachedUserPolicies" {
			t.Fatalf("unexpected action: %s", got)
		}
		if got := values.Get("UserName"); got != "alice" {
			t.Fatalf("unexpected user: %s", got)
		}
		_, _ = w.Write([]byte(`
<ListAttachedUserPoliciesResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/">
  <ListAttachedUserPoliciesResult>
    <AttachedPolicies>
      <member>
        <PolicyName>AdministratorAccess</PolicyName>
        <PolicyArn>arn:aws:iam::aws:policy/AdministratorAccess</PolicyArn>
      </member>
      <member>
        <PolicyName>ReadOnlyAccess</PolicyName>
        <PolicyArn>arn:aws:iam::aws:policy/ReadOnlyAccess</PolicyArn>
      </member>
    </AttachedPolicies>
    <IsTruncated>false</IsTruncated>
  </ListAttachedUserPoliciesResult>
  <ResponseMetadata>
    <RequestId>req-list-attached</RequestId>
  </ResponseMetadata>
</ListAttachedUserPoliciesResponse>`))
	}))
	defer server.Close()

	driver := newRoleBindingTestDriver(server.URL)
	got, err := driver.ListRoleBindings(context.Background(), "alice")
	if err != nil {
		t.Fatalf("ListRoleBindings: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 bindings, got %d", len(got))
	}
	if got[0].Role != "AdministratorAccess" || got[0].Scope != "arn:aws:iam::aws:policy/AdministratorAccess" {
		t.Errorf("unexpected first binding: %+v", got[0])
	}
	if got[1].Role != "ReadOnlyAccess" {
		t.Errorf("unexpected second binding: %+v", got[1])
	}
	if got[0].Principal != "alice" {
		t.Errorf("expected principal 'alice', got %q", got[0].Principal)
	}
}

func TestListRoleBindingsRejectsEmptyPrincipal(t *testing.T) {
	driver := newRoleBindingTestDriver("http://example.invalid")
	if _, err := driver.ListRoleBindings(context.Background(), "  "); err == nil {
		t.Fatalf("expected error for empty principal")
	}
}

func TestAttachPolicySendsExpectedForm(t *testing.T) {
	var captured url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = parseRoleBindingForm(t, r)
		_, _ = w.Write([]byte(`<AttachUserPolicyResponse><ResponseMetadata><RequestId>r1</RequestId></ResponseMetadata></AttachUserPolicyResponse>`))
	}))
	defer server.Close()

	driver := newRoleBindingTestDriver(server.URL)
	if err := driver.AttachPolicy(context.Background(), "alice", "arn:aws:iam::aws:policy/AdministratorAccess"); err != nil {
		t.Fatalf("AttachPolicy: %v", err)
	}
	if captured.Get("Action") != "AttachUserPolicy" {
		t.Errorf("unexpected action: %s", captured.Get("Action"))
	}
	if captured.Get("UserName") != "alice" {
		t.Errorf("unexpected user: %s", captured.Get("UserName"))
	}
	if captured.Get("PolicyArn") != "arn:aws:iam::aws:policy/AdministratorAccess" {
		t.Errorf("unexpected policy ARN: %s", captured.Get("PolicyArn"))
	}
}

func TestDetachPolicyPropagatesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`
<ErrorResponse>
  <Error>
    <Type>Sender</Type>
    <Code>NoSuchEntity</Code>
    <Message>Policy is not attached</Message>
  </Error>
  <RequestId>r1</RequestId>
</ErrorResponse>`))
	}))
	defer server.Close()

	driver := newRoleBindingTestDriver(server.URL)
	err := driver.DetachPolicy(context.Background(), "alice", "arn:aws:iam::aws:policy/Never")
	if err == nil {
		t.Fatalf("expected error from DetachPolicy")
	}
	if !strings.Contains(err.Error(), "NoSuchEntity") {
		t.Errorf("expected NoSuchEntity in error, got %v", err)
	}
}

func TestResolvePolicyARN(t *testing.T) {
	cases := map[string]string{
		"":                       "",
		"AdministratorAccess":    "arn:aws:iam::aws:policy/AdministratorAccess",
		"ReadOnlyAccess":         "arn:aws:iam::aws:policy/ReadOnlyAccess",
		"arn:aws:iam::aws:policy/Custom": "arn:aws:iam::aws:policy/Custom",
		"arn:aws-cn:iam::123456:policy/Tenant": "arn:aws-cn:iam::123456:policy/Tenant",
	}
	for in, want := range cases {
		if got := ResolvePolicyARN(in); got != want {
			t.Errorf("ResolvePolicyARN(%q) = %q, want %q", in, got, want)
		}
	}
}
