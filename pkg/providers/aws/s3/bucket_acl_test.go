package s3

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
)

func TestGetBucketAclParsesGrants(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if _, ok := r.URL.Query()["acl"]; !ok {
			t.Fatalf("missing ?acl query")
		}
		_, _ = w.Write([]byte(`
<AccessControlPolicy xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Owner><ID>owner-123</ID><DisplayName>ctk-demo</DisplayName></Owner>
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

	client := newS3DriverTestClient(server.URL)
	out, err := client.GetBucketAcl(context.Background(), "us-east-1", "ctk-demo")
	if err != nil {
		t.Fatalf("GetBucketAcl: %v", err)
	}
	if S3CannedACLFromGrants(out) != S3ACLPublicRead {
		t.Errorf("expected public-read summary, got %q", S3CannedACLFromGrants(out))
	}
}

func TestPutBucketAclSendsCannedHeader(t *testing.T) {
	var sawACL string
	var sawQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("expected PUT, got %s", r.Method)
		}
		sawACL = r.Header.Get("x-amz-acl")
		sawQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newS3DriverTestClient(server.URL)
	if err := client.PutBucketAcl(context.Background(), "us-east-1", "ctk-demo", S3ACLPublicRead); err != nil {
		t.Fatalf("PutBucketAcl: %v", err)
	}
	if sawACL != "public-read" {
		t.Errorf("expected x-amz-acl=public-read, got %q", sawACL)
	}
	if !strings.HasPrefix(sawQuery, "acl") {
		t.Errorf("expected ?acl, got %q", sawQuery)
	}
}

func TestExposeBucketDefaultsToPublicReadAndClearsBPA(t *testing.T) {
	var sawDelete bool
	var sawACLHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/":
			_, _ = w.Write([]byte(`
<ListAllMyBucketsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Buckets><Bucket><Name>ctk-demo</Name><BucketRegion>us-east-1</BucketRegion></Bucket></Buckets>
</ListAllMyBucketsResult>`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "ctk-demo") && r.URL.Query().Has("location"):
			_, _ = w.Write([]byte(`<LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/">us-east-1</LocationConstraint>`))
		case r.Method == http.MethodDelete && r.URL.Query().Has("publicAccessBlock"):
			sawDelete = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPut && r.URL.Query().Has("acl"):
			sawACLHeader = r.Header.Get("x-amz-acl")
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
		}
	}))
	defer server.Close()

	driver := &Driver{Client: newS3DriverTestClient(server.URL), DefaultRegion: "us-east-1"}
	applied, err := driver.ExposeBucket(context.Background(), "ctk-demo", "")
	if err != nil {
		t.Fatalf("ExposeBucket: %v", err)
	}
	if applied != S3ACLPublicRead {
		t.Errorf("expected public-read, got %q", applied)
	}
	if !sawDelete {
		t.Errorf("expected DeletePublicAccessBlock to be called")
	}
	if sawACLHeader != "public-read" {
		t.Errorf("expected x-amz-acl=public-read, got %q", sawACLHeader)
	}
}

func TestS3CannedACLFromGrants(t *testing.T) {
	cases := []struct {
		name string
		in   api.GetBucketAclOutput
		want string
	}{
		{
			name: "no public grants",
			in:   api.GetBucketAclOutput{Grants: []api.S3Grant{{Permission: "FULL_CONTROL", GranteeID: "owner"}}},
			want: S3ACLPrivate,
		},
		{
			name: "public read",
			in: api.GetBucketAclOutput{Grants: []api.S3Grant{{
				Permission: "READ",
				GranteeURI: "http://acs.amazonaws.com/groups/global/AllUsers",
			}}},
			want: S3ACLPublicRead,
		},
		{
			name: "public read+write",
			in: api.GetBucketAclOutput{Grants: []api.S3Grant{
				{Permission: "READ", GranteeURI: "http://acs.amazonaws.com/groups/global/AllUsers"},
				{Permission: "WRITE", GranteeURI: "http://acs.amazonaws.com/groups/global/AllUsers"},
			}},
			want: S3ACLPublicReadWrite,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := S3CannedACLFromGrants(c.in); got != c.want {
				t.Errorf("S3CannedACLFromGrants() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestNormalizeS3ACL(t *testing.T) {
	cases := map[string]string{
		"":                    S3ACLPrivate,
		"private":             S3ACLPrivate,
		"public-read":         S3ACLPublicRead,
		"PublicRead":          S3ACLPublicRead,
		"writable":            S3ACLPublicReadWrite,
		"authenticated-read":  S3ACLAuthenticatedRead,
	}
	for in, want := range cases {
		if got := NormalizeS3ACL(in); got != want {
			t.Errorf("NormalizeS3ACL(%q) = %q, want %q", in, got, want)
		}
	}
}
