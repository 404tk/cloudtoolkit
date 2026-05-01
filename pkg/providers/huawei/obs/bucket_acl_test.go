package obs

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
)

type aclTransport func(*http.Request) (*http.Response, error)

func (fn aclTransport) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

func newACLClient(fn aclTransport) *Client {
	return NewClient(
		auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", "cn-north-4", false),
		WithHTTPClient(&http.Client{Transport: fn}),
		WithRetryPolicy(noopRetryPolicy{}),
		WithClock(func() time.Time { return time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC) }),
	)
}

func TestGetBucketACLCollapsesGrantsToCanned(t *testing.T) {
	t.Parallel()

	const body = `<?xml version="1.0" encoding="UTF-8"?>
<AccessControlPolicy>
  <Owner><ID>1001</ID><DisplayName>ctk-demo</DisplayName></Owner>
  <AccessControlList>
    <Grant>
      <Grantee xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="Group">
        <URI>Everyone</URI>
      </Grantee>
      <Permission>READ</Permission>
    </Grant>
  </AccessControlList>
</AccessControlPolicy>`

	client := newACLClient(aclTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.RawQuery != "acl=" {
			t.Fatalf("expected ?acl, got %q", r.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    r,
		}, nil
	}))
	got, err := client.GetBucketACL(context.Background(), "ctk-demo", "cn-north-4")
	if err != nil {
		t.Fatalf("GetBucketACL: %v", err)
	}
	if got != OBSACLPublicRead {
		t.Errorf("expected public-read, got %q", got)
	}
}

func TestPutBucketACLSendsHeaderAndQuery(t *testing.T) {
	t.Parallel()

	var sawACL string
	var sawQuery string
	client := newACLClient(aclTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPut {
			t.Fatalf("expected PUT, got %s", r.Method)
		}
		sawACL = r.Header.Get("x-obs-acl")
		sawQuery = r.URL.RawQuery
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    r,
		}, nil
	}))
	if err := client.PutBucketACL(context.Background(), "ctk-demo", "cn-north-4", OBSACLPublicRead); err != nil {
		t.Fatalf("PutBucketACL: %v", err)
	}
	if sawACL != "public-read" {
		t.Errorf("expected x-obs-acl=public-read, got %q", sawACL)
	}
	if sawQuery != "acl=" {
		t.Errorf("expected ?acl, got %q", sawQuery)
	}
}

func TestNormalizeOBSACL(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":                          OBSACLPrivate,
		"private":                   OBSACLPrivate,
		"public-read":               OBSACLPublicRead,
		"PublicRead":                OBSACLPublicRead,
		"writable":                  OBSACLPublicReadWrite,
		"public-read-delivered":     OBSACLPublicReadDelivered,
		"public-read-write-delivered": OBSACLPublicReadWriteDelivered,
	}
	for in, want := range cases {
		if got := NormalizeOBSACL(in); got != want {
			t.Errorf("NormalizeOBSACL(%q) = %q, want %q", in, got, want)
		}
	}
}
