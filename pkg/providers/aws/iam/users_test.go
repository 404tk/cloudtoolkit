package iam

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/aws/auth"
)

func TestDriverListUsersMapsLoginAndPolicies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values := mustParseIAMBodyValues(t, r)
		switch values.Get("Action") {
		case "ListUsers":
			_, _ = w.Write([]byte(`
<ListUsersResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/">
  <ListUsersResult>
    <Users>
      <member>
        <UserName>alice</UserName>
        <UserId>AIDAALICE</UserId>
        <CreateDate>2026-04-18T10:00:00Z</CreateDate>
        <PasswordLastUsed>2026-04-18T11:00:00Z</PasswordLastUsed>
      </member>
      <member>
        <UserName>bob</UserName>
        <UserId>AIDABOB</UserId>
        <CreateDate>2026-04-18T09:00:00Z</CreateDate>
      </member>
    </Users>
    <IsTruncated>false</IsTruncated>
  </ListUsersResult>
  <ResponseMetadata>
    <RequestId>req-list-users</RequestId>
  </ResponseMetadata>
</ListUsersResponse>`))
		case "GetLoginProfile":
			if got := values.Get("UserName"); got != "bob" {
				t.Fatalf("unexpected username for GetLoginProfile: %s", got)
			}
			_, _ = w.Write([]byte(`
<GetLoginProfileResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/">
  <GetLoginProfileResult>
    <LoginProfile>
      <CreateDate>2026-04-18T12:00:00Z</CreateDate>
      <PasswordResetRequired>false</PasswordResetRequired>
    </LoginProfile>
  </GetLoginProfileResult>
  <ResponseMetadata>
    <RequestId>req-login-profile</RequestId>
  </ResponseMetadata>
</GetLoginProfileResponse>`))
		case "ListAttachedUserPolicies":
			switch values.Get("UserName") {
			case "alice":
				_, _ = w.Write([]byte(`
<ListAttachedUserPoliciesResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/">
  <ListAttachedUserPoliciesResult>
    <AttachedPolicies>
      <member>
        <PolicyName>AdministratorAccess</PolicyName>
        <PolicyArn>arn:aws:iam::aws:policy/AdministratorAccess</PolicyArn>
      </member>
    </AttachedPolicies>
    <IsTruncated>false</IsTruncated>
  </ListAttachedUserPoliciesResult>
  <ResponseMetadata>
    <RequestId>req-policy-alice</RequestId>
  </ResponseMetadata>
</ListAttachedUserPoliciesResponse>`))
			case "bob":
				if values.Get("Marker") == "" {
					_, _ = w.Write([]byte(`
<ListAttachedUserPoliciesResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/">
  <ListAttachedUserPoliciesResult>
    <AttachedPolicies>
      <member>
        <PolicyName>ReadOnlyAccess</PolicyName>
        <PolicyArn>arn:aws:iam::aws:policy/ReadOnlyAccess</PolicyArn>
      </member>
    </AttachedPolicies>
    <IsTruncated>true</IsTruncated>
    <Marker>page-2</Marker>
  </ListAttachedUserPoliciesResult>
  <ResponseMetadata>
    <RequestId>req-policy-bob-1</RequestId>
  </ResponseMetadata>
</ListAttachedUserPoliciesResponse>`))
				} else {
					_, _ = w.Write([]byte(`
<ListAttachedUserPoliciesResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/">
  <ListAttachedUserPoliciesResult>
    <AttachedPolicies>
      <member>
        <PolicyName>SupportUser</PolicyName>
        <PolicyArn>arn:aws:iam::aws:policy/SupportUser</PolicyArn>
      </member>
    </AttachedPolicies>
    <IsTruncated>false</IsTruncated>
  </ListAttachedUserPoliciesResult>
  <ResponseMetadata>
    <RequestId>req-policy-bob-2</RequestId>
  </ResponseMetadata>
</ListAttachedUserPoliciesResponse>`))
				}
			default:
				t.Fatalf("unexpected username for ListAttachedUserPolicies: %s", values.Get("UserName"))
			}
		default:
			t.Fatalf("unexpected action: %s", values.Get("Action"))
		}
	}))
	defer server.Close()

	driver := &Driver{
		Client: newIAMDriverTestClient(server.URL),
		Region: "us-east-1",
	}

	got, err := driver.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("unexpected user count: %d", len(got))
	}
	if !got[0].EnableLogin || got[0].Policies != "AdministratorAccess" {
		t.Fatalf("unexpected first user: %+v", got[0])
	}
	if !got[1].EnableLogin || got[1].Policies != "ReadOnlyAccess\nSupportUser" {
		t.Fatalf("unexpected second user: %+v", got[1])
	}
}

func TestDriverListUsersUsesDefaultRegionForChinaAll(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values := mustParseIAMBodyValues(t, r)
		if got := values.Get("Action"); got != "ListUsers" {
			t.Fatalf("unexpected action: %s", got)
		}
		if got := signingRegionFromAuthorization(t, r.Header.Get("Authorization")); got != "cn-north-1" {
			t.Fatalf("unexpected signing region: %s", got)
		}
		_, _ = w.Write([]byte(`
<ListUsersResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/">
  <ListUsersResult>
    <Users></Users>
    <IsTruncated>false</IsTruncated>
  </ListUsersResult>
  <ResponseMetadata>
    <RequestId>req-list-users-cn</RequestId>
  </ResponseMetadata>
</ListUsersResponse>`))
	}))
	defer server.Close()

	driver := &Driver{
		Client:        newIAMDriverTestClient(server.URL),
		Region:        "all",
		DefaultRegion: "cn-northwest-1",
	}

	got, err := driver.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("unexpected user count: %d", len(got))
	}
}

func TestDriverRequestRegionFallsBackToDefaultRegion(t *testing.T) {
	driver := &Driver{
		Region:        "all",
		DefaultRegion: "cn-northwest-1",
	}
	if got := driver.requestRegion(); got != "cn-northwest-1" {
		t.Fatalf("requestRegion() = %q, want %q", got, "cn-northwest-1")
	}
}

func TestEntityErrorHelpers(t *testing.T) {
	if !isEntityAlreadyExists(&api.APIError{Code: "EntityAlreadyExists"}) {
		t.Fatal("expected entity already exists helper to match")
	}
	if isEntityAlreadyExists(&api.APIError{Code: "NoSuchEntity"}) {
		t.Fatal("unexpected entity already exists match")
	}
	if !isNoSuchEntity(&api.APIError{Code: "NoSuchEntity"}) {
		t.Fatal("expected no such entity helper to match")
	}
}

func newIAMDriverTestClient(baseURL string) *api.Client {
	return api.NewClient(
		auth.New("AKID", "SECRET", ""),
		api.WithBaseURL(baseURL),
		api.WithClock(func() time.Time { return time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC) }),
		api.WithRetryPolicy(api.RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)
}

func mustParseIAMBodyValues(t *testing.T, r *http.Request) url.Values {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	_ = r.Body.Close()
	values, err := url.ParseQuery(string(body))
	if err != nil {
		t.Fatalf("parse body: %v", err)
	}
	return values
}

func signingRegionFromAuthorization(t *testing.T, authorization string) string {
	t.Helper()
	const prefix = "Credential="
	start := strings.Index(authorization, prefix)
	if start < 0 {
		t.Fatalf("missing credential scope: %s", authorization)
	}
	scope := authorization[start+len(prefix):]
	if end := strings.Index(scope, ","); end >= 0 {
		scope = scope[:end]
	}
	parts := strings.Split(scope, "/")
	if len(parts) < 5 {
		t.Fatalf("invalid credential scope: %s", authorization)
	}
	return parts[2]
}
