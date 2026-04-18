package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/auth"
)

func TestNormalizeServiceRegionAndDefaultHostForIAM(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantRegion string
		wantHost   string
	}{
		{name: "standard partition", input: "ap-southeast-1", wantRegion: "us-east-1", wantHost: "iam.amazonaws.com"},
		{name: "china partition", input: "cn-northwest-1", wantRegion: "cn-north-1", wantHost: "iam.cn-north-1.amazonaws.com.cn"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRegion := normalizeServiceRegion("iam", tt.input)
			if gotRegion != tt.wantRegion {
				t.Fatalf("normalizeServiceRegion() = %q, want %q", gotRegion, tt.wantRegion)
			}
			if gotHost := defaultHost("iam", gotRegion); gotHost != tt.wantHost {
				t.Fatalf("defaultHost() = %q, want %q", gotHost, tt.wantHost)
			}
		})
	}
}

func TestIAMListUsersParsesUsersAndMarker(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values := mustParseBodyValues(t, r)
		if got := values.Get("Action"); got != "ListUsers" {
			t.Fatalf("unexpected action: %s", got)
		}
		if got := values.Get("Version"); got != iamAPIVersion {
			t.Fatalf("unexpected version: %s", got)
		}
		authz := r.Header.Get("Authorization")
		if !strings.Contains(authz, "Credential=AKID/20260418/us-east-1/iam/aws4_request") {
			t.Fatalf("unexpected authorization header: %s", authz)
		}
		_, _ = w.Write([]byte(`
<ListUsersResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/">
  <ListUsersResult>
    <Users>
      <member>
        <UserName>alice</UserName>
        <UserId>AIDAALICE</UserId>
        <Arn>arn:aws:iam::123456789012:user/alice</Arn>
        <CreateDate>2026-04-18T10:00:00Z</CreateDate>
        <PasswordLastUsed>2026-04-18T11:00:00Z</PasswordLastUsed>
      </member>
      <member>
        <UserName>bob</UserName>
        <UserId>AIDABOB</UserId>
        <Arn>arn:aws:iam::123456789012:user/bob</Arn>
        <CreateDate>2026-04-18T09:00:00Z</CreateDate>
      </member>
    </Users>
    <IsTruncated>true</IsTruncated>
    <Marker>next-marker</Marker>
  </ListUsersResult>
  <ResponseMetadata>
    <RequestId>req-list-users</RequestId>
  </ResponseMetadata>
</ListUsersResponse>`))
	}))
	defer server.Close()

	client := newIAMTestClient(server.URL)
	got, err := client.ListUsers(context.Background(), "ap-southeast-1", "")
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if got.RequestID != "req-list-users" || !got.IsTruncated || got.Marker != "next-marker" {
		t.Fatalf("unexpected paging metadata: %+v", got)
	}
	if len(got.Users) != 2 {
		t.Fatalf("unexpected user count: %d", len(got.Users))
	}
	if got.Users[0].UserName != "alice" || got.Users[0].PasswordLastUsed == nil || got.Users[1].PasswordLastUsed != nil {
		t.Fatalf("unexpected users: %+v", got.Users)
	}
}

func TestIAMGetLoginProfileParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values := mustParseBodyValues(t, r)
		if got := values.Get("Action"); got != "GetLoginProfile" {
			t.Fatalf("unexpected action: %s", got)
		}
		if got := values.Get("UserName"); got != "demo" {
			t.Fatalf("unexpected username: %s", got)
		}
		_, _ = w.Write([]byte(`
<GetLoginProfileResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/">
  <GetLoginProfileResult>
    <LoginProfile>
      <CreateDate>2026-04-18T08:00:00Z</CreateDate>
      <PasswordResetRequired>false</PasswordResetRequired>
    </LoginProfile>
  </GetLoginProfileResult>
  <ResponseMetadata>
    <RequestId>req-login-profile</RequestId>
  </ResponseMetadata>
</GetLoginProfileResponse>`))
	}))
	defer server.Close()

	client := newIAMTestClient(server.URL)
	got, err := client.GetLoginProfile(context.Background(), "us-east-1", "demo")
	if err != nil {
		t.Fatalf("GetLoginProfile() error = %v", err)
	}
	if got.RequestID != "req-login-profile" || got.CreateDate == nil || got.PasswordResetRequired {
		t.Fatalf("unexpected login profile: %+v", got)
	}
}

func TestIAMListAttachedUserPoliciesParsesPoliciesAndMarker(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values := mustParseBodyValues(t, r)
		if got := values.Get("Action"); got != "ListAttachedUserPolicies" {
			t.Fatalf("unexpected action: %s", got)
		}
		if got := values.Get("UserName"); got != "demo" {
			t.Fatalf("unexpected username: %s", got)
		}
		if got := values.Get("Marker"); got != "page-2" {
			t.Fatalf("unexpected marker: %s", got)
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
    <RequestId>req-policies</RequestId>
  </ResponseMetadata>
</ListAttachedUserPoliciesResponse>`))
	}))
	defer server.Close()

	client := newIAMTestClient(server.URL)
	got, err := client.ListAttachedUserPolicies(context.Background(), "us-east-1", "demo", "page-2")
	if err != nil {
		t.Fatalf("ListAttachedUserPolicies() error = %v", err)
	}
	if got.RequestID != "req-policies" || len(got.Policies) != 2 || got.Policies[0].PolicyName != "AdministratorAccess" {
		t.Fatalf("unexpected policies: %+v", got)
	}
}

func TestIAMCreateUserParsesArn(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values := mustParseBodyValues(t, r)
		if got := values.Get("Action"); got != "CreateUser" {
			t.Fatalf("unexpected action: %s", got)
		}
		if got := values.Get("UserName"); got != "demo" {
			t.Fatalf("unexpected username: %s", got)
		}
		authz := r.Header.Get("Authorization")
		if !strings.Contains(authz, "/us-east-1/iam/aws4_request") {
			t.Fatalf("unexpected authorization header: %s", authz)
		}
		_, _ = w.Write([]byte(`
<CreateUserResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/">
  <CreateUserResult>
    <User>
      <Arn>arn:aws:iam::123456789012:user/demo</Arn>
    </User>
  </CreateUserResult>
  <ResponseMetadata>
    <RequestId>req-create-user</RequestId>
  </ResponseMetadata>
</CreateUserResponse>`))
	}))
	defer server.Close()

	client := newIAMTestClient(server.URL)
	got, err := client.CreateUser(context.Background(), "ap-southeast-1", "demo")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if got.Arn != "arn:aws:iam::123456789012:user/demo" || got.RequestID != "req-create-user" {
		t.Fatalf("unexpected create user response: %+v", got)
	}
}

func TestIAMMutationActionsSendExpectedBody(t *testing.T) {
	tests := []struct {
		name     string
		call     func(*Client) error
		action   string
		expected map[string]string
	}{
		{
			name: "create login profile",
			call: func(c *Client) error {
				return c.CreateLoginProfile(context.Background(), "us-east-1", "demo", "SecretPassw0rd!")
			},
			action: "CreateLoginProfile",
			expected: map[string]string{
				"UserName": "demo",
				"Password": "SecretPassw0rd!",
			},
		},
		{
			name: "attach user policy",
			call: func(c *Client) error {
				return c.AttachUserPolicy(context.Background(), "us-east-1", "demo", "arn:aws:iam::aws:policy/AdministratorAccess")
			},
			action: "AttachUserPolicy",
			expected: map[string]string{
				"UserName":  "demo",
				"PolicyArn": "arn:aws:iam::aws:policy/AdministratorAccess",
			},
		},
		{
			name: "detach user policy",
			call: func(c *Client) error {
				return c.DetachUserPolicy(context.Background(), "us-east-1", "demo", "arn:aws:iam::aws:policy/AdministratorAccess")
			},
			action: "DetachUserPolicy",
			expected: map[string]string{
				"UserName":  "demo",
				"PolicyArn": "arn:aws:iam::aws:policy/AdministratorAccess",
			},
		},
		{
			name:   "delete login profile",
			call:   func(c *Client) error { return c.DeleteLoginProfile(context.Background(), "us-east-1", "demo") },
			action: "DeleteLoginProfile",
			expected: map[string]string{
				"UserName": "demo",
			},
		},
		{
			name:   "delete user",
			call:   func(c *Client) error { return c.DeleteUser(context.Background(), "us-east-1", "demo") },
			action: "DeleteUser",
			expected: map[string]string{
				"UserName": "demo",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				values := mustParseBodyValues(t, r)
				if got := values.Get("Action"); got != tt.action {
					t.Fatalf("unexpected action: %s", got)
				}
				if got := values.Get("Version"); got != iamAPIVersion {
					t.Fatalf("unexpected version: %s", got)
				}
				for key, want := range tt.expected {
					if got := values.Get(key); got != want {
						t.Fatalf("unexpected %s: %s", key, got)
					}
				}
				_, _ = w.Write([]byte(`
<ResponseMetadata>
  <RequestId>req-mutation</RequestId>
</ResponseMetadata>`))
			}))
			defer server.Close()

			client := newIAMTestClient(server.URL)
			if err := tt.call(client); err != nil {
				t.Fatalf("call error = %v", err)
			}
		})
	}
}

func newIAMTestClient(baseURL string) *Client {
	return NewClient(
		auth.New("AKID", "SECRET", ""),
		WithBaseURL(baseURL),
		WithClock(func() time.Time { return time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC) }),
		WithRetryPolicy(RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)
}

func mustParseBodyValues(t *testing.T, r *http.Request) url.Values {
	t.Helper()
	values, err := url.ParseQuery(readBody(t, r))
	if err != nil {
		t.Fatalf("parse request body: %v", err)
	}
	return values
}
