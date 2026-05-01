package oss

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	awsapi "github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/auth"
)

func newACLClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	return NewClient(
		auth.New("AKID", "SECRET", ""),
		awsapi.WithHTTPClient(rewriteHostClient(baseURL)),
		awsapi.WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
		awsapi.WithRetryPolicy(awsapi.RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)
}

func TestGetBucketAclParsesGrants(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if !r.URL.Query().Has("acl") {
			t.Fatalf("missing ?acl query")
		}
		_, _ = w.Write([]byte(`
<AccessControlPolicy>
  <Owner><ID>owner-1</ID><DisplayName>ctk-demo</DisplayName></Owner>
  <AccessControlList>
    <Grant>
      <Grantee xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="Group">
        <URI>http://acs.amazonaws.com/groups/global/AllUsers</URI>
      </Grantee>
      <Permission>READ</Permission>
    </Grant>
  </AccessControlList>
</AccessControlPolicy>`))
	}))
	defer server.Close()

	client := newACLClient(t, server.URL)
	out, err := client.GetBucketAcl(context.Background(), "ctk-demo", "cn-north-1")
	if err != nil {
		t.Fatalf("GetBucketAcl: %v", err)
	}
	if got := CannedACLFromGrants(out); got != OSSACLPublicRead {
		t.Errorf("expected public-read, got %q", got)
	}
}

func TestPutBucketAclSendsCannedHeader(t *testing.T) {
	var sawACL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("expected PUT, got %s", r.Method)
		}
		sawACL = r.Header.Get("x-amz-acl")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newACLClient(t, server.URL)
	if err := client.PutBucketAcl(context.Background(), "ctk-demo", "cn-north-1", OSSACLPublicRead); err != nil {
		t.Fatalf("PutBucketAcl: %v", err)
	}
	if sawACL != "public-read" {
		t.Errorf("expected x-amz-acl=public-read, got %q", sawACL)
	}
}

func TestNormalizeOSSACL(t *testing.T) {
	cases := map[string]string{
		"":                    OSSACLPrivate,
		"public-read":         OSSACLPublicRead,
		"PublicRead":          OSSACLPublicRead,
		"writable":            OSSACLPublicReadWrite,
		"authenticated-read":  OSSACLAuthenticatedRead,
	}
	for in, want := range cases {
		if got := NormalizeOSSACL(in); got != want {
			t.Errorf("NormalizeOSSACL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCannedACLFromGrants(t *testing.T) {
	out := BucketACLOutput{
		Grants: []Grant{
			{Permission: "READ", GranteeURI: "http://acs.amazonaws.com/groups/global/AllUsers"},
			{Permission: "WRITE", GranteeURI: "http://acs.amazonaws.com/groups/global/AllUsers"},
		},
	}
	if got := CannedACLFromGrants(out); got != OSSACLPublicReadWrite {
		t.Errorf("expected public-read-write, got %q", got)
	}
	priv := BucketACLOutput{Grants: []Grant{{Permission: "FULL_CONTROL", GranteeID: "owner"}}}
	if got := CannedACLFromGrants(priv); got != OSSACLPrivate {
		t.Errorf("expected private, got %q", got)
	}
	_ = strings.Repeat
}
