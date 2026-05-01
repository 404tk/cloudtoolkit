package cos

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
)

func newACLClient(t *testing.T, fn roundTripFunc) *Client {
	t.Helper()
	return NewClient(
		auth.New("AKID", "SECRET", ""),
		WithHTTPClient(&http.Client{Transport: fn}),
		WithRetryPolicy(api.RetryPolicy{MaxAttempts: 1}),
		WithClock(func() time.Time { return time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC) }),
	)
}

func TestGetBucketACLCollapsesGrants(t *testing.T) {
	t.Parallel()

	const body = `<?xml version="1.0" encoding="UTF-8"?>
<AccessControlPolicy>
  <Owner><ID>1001</ID><DisplayName>ctk-demo</DisplayName></Owner>
  <AccessControlList>
    <Grant>
      <Grantee xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="Group">
        <URI>http://cam.qcloud.com/groups/global/AllUsers</URI>
      </Grantee>
      <Permission>READ</Permission>
    </Grant>
  </AccessControlList>
</AccessControlPolicy>`

	client := newACLClient(t, func(r *http.Request) (*http.Response, error) {
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
	})
	got, err := client.GetBucketACL(context.Background(), "ctk-demo", "ap-shanghai")
	if err != nil {
		t.Fatalf("GetBucketACL: %v", err)
	}
	if got != COSACLPublicRead {
		t.Errorf("expected public-read, got %q", got)
	}
}

func TestPutBucketACLSendsHeaderAndQuery(t *testing.T) {
	t.Parallel()

	var sawACL string
	var sawQuery string
	client := newACLClient(t, func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPut {
			t.Fatalf("expected PUT, got %s", r.Method)
		}
		sawACL = r.Header.Get("x-cos-acl")
		sawQuery = r.URL.RawQuery
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    r,
		}, nil
	})
	if err := client.PutBucketACL(context.Background(), "ctk-demo", "ap-shanghai", COSACLPublicRead); err != nil {
		t.Fatalf("PutBucketACL: %v", err)
	}
	if sawACL != "public-read" {
		t.Errorf("expected x-cos-acl=public-read, got %q", sawACL)
	}
	if sawQuery != "acl=" {
		t.Errorf("expected ?acl, got %q", sawQuery)
	}
}

func TestNormalizeCOSACL(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":                   COSACLPrivate,
		"private":            COSACLPrivate,
		"public-read":        COSACLPublicRead,
		"PublicRead":         COSACLPublicRead,
		"writable":           COSACLPublicReadWrite,
		"authenticated-read": COSACLAuthenticatedRead,
	}
	for in, want := range cases {
		if got := NormalizeCOSACL(in); got != want {
			t.Errorf("NormalizeCOSACL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCollapseGrants(t *testing.T) {
	t.Parallel()

	out := BucketACLResponse{}
	out.AccessControlList.Grant = append(out.AccessControlList.Grant, struct {
		Grantee struct {
			Type string `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr"`
			ID   string `xml:"ID"`
			URI  string `xml:"URI"`
		} `xml:"Grantee"`
		Permission string `xml:"Permission"`
	}{
		Grantee: struct {
			Type string `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr"`
			ID   string `xml:"ID"`
			URI  string `xml:"URI"`
		}{Type: "Group", URI: "http://cam.qcloud.com/groups/global/AllUsers"},
		Permission: "FULL_CONTROL",
	})
	if got := CollapseGrants(out); got != COSACLPublicReadWrite {
		t.Errorf("expected public-read-write, got %q", got)
	}
}
