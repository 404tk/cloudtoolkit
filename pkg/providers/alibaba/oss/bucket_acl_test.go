package oss

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
)

// fakeOSSDriver wires a Driver against an in-process roundTripFunc so the ACL
// tests don't need real HTTP endpoints.
func fakeOSSDriver(t *testing.T, region string, fn roundTripFunc) Driver {
	t.Helper()
	return Driver{
		Cred:   aliauth.New("ak", "sk", ""),
		Region: region,
		clientOptions: []Option{
			WithHTTPClient(&http.Client{Transport: fn}),
			WithRetryPolicy(api.RetryPolicy{MaxAttempts: 1}),
			WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
		},
	}
}

func TestGetBucketACLParsesGrant(t *testing.T) {
	t.Parallel()

	const body = `<?xml version="1.0" encoding="UTF-8"?>
<AccessControlPolicy>
  <Owner><ID>10001</ID><DisplayName>ctk-demo</DisplayName></Owner>
  <AccessControlList><Grant>public-read</Grant></AccessControlList>
</AccessControlPolicy>`

	driver := fakeOSSDriver(t, "cn-hangzhou", func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if got := req.URL.RawQuery; got != "acl=" {
			t.Fatalf("expected ?acl, got %q", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/xml"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})
	client, err := driver.NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	got, err := client.GetBucketACL(context.Background(), "ctk-demo", "cn-hangzhou")
	if err != nil {
		t.Fatalf("GetBucketACL: %v", err)
	}
	if got != "public-read" {
		t.Errorf("unexpected grant: %q", got)
	}
}

func TestPutBucketACLSendsHeaderAndQuery(t *testing.T) {
	t.Parallel()

	var sawACL string
	var sawQuery string
	var sawMethod string
	driver := fakeOSSDriver(t, "cn-hangzhou", func(req *http.Request) (*http.Response, error) {
		sawACL = req.Header.Get("x-oss-acl")
		sawQuery = req.URL.RawQuery
		sawMethod = req.Method
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/xml"}},
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    req,
		}, nil
	})
	client, err := driver.NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if err := client.PutBucketACL(context.Background(), "ctk-demo", "cn-hangzhou", OSSACLPublicRead); err != nil {
		t.Fatalf("PutBucketACL: %v", err)
	}
	if sawMethod != http.MethodPut {
		t.Errorf("expected PUT, got %s", sawMethod)
	}
	if sawACL != "public-read" {
		t.Errorf("expected x-oss-acl=public-read, got %q", sawACL)
	}
	if sawQuery != "acl=" {
		t.Errorf("expected ?acl, got %q", sawQuery)
	}
}

func TestExposeBucketDefaultsToPublicRead(t *testing.T) {
	t.Parallel()

	var sawACL string
	driver := fakeOSSDriver(t, "cn-hangzhou", func(req *http.Request) (*http.Response, error) {
		sawACL = req.Header.Get("x-oss-acl")
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/xml"}},
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    req,
		}, nil
	})
	applied, err := driver.ExposeBucket(context.Background(), "ctk-demo", "")
	if err != nil {
		t.Fatalf("ExposeBucket: %v", err)
	}
	if applied != OSSACLPublicRead {
		t.Errorf("expected applied=public-read, got %q", applied)
	}
	if sawACL != "public-read" {
		t.Errorf("expected sent x-oss-acl=public-read, got %q", sawACL)
	}
}

func TestUnexposeBucketAppliesPrivate(t *testing.T) {
	t.Parallel()

	var sawACL string
	driver := fakeOSSDriver(t, "cn-hangzhou", func(req *http.Request) (*http.Response, error) {
		sawACL = req.Header.Get("x-oss-acl")
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/xml"}},
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    req,
		}, nil
	})
	if err := driver.UnexposeBucket(context.Background(), "ctk-demo"); err != nil {
		t.Fatalf("UnexposeBucket: %v", err)
	}
	if sawACL != OSSACLPrivate {
		t.Errorf("expected x-oss-acl=private, got %q", sawACL)
	}
}

func TestNormalizeOSSACL(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":                  OSSACLPrivate,
		" private ":         OSSACLPrivate,
		"public-read":       OSSACLPublicRead,
		"PublicRead":        OSSACLPublicRead,
		"Blob":              OSSACLPublicRead,
		"public-read-write": OSSACLPublicReadWrite,
		"writable":          OSSACLPublicReadWrite,
	}
	for in, want := range cases {
		if got := NormalizeOSSACL(in); got != want {
			t.Errorf("NormalizeOSSACL(%q) = %q, want %q", in, got, want)
		}
	}
}
