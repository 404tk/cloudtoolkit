package tos

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	volcapi "github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/auth"
)

func newTOSACLDriver(t *testing.T, region string, fn roundTripFunc) Driver {
	t.Helper()
	return Driver{
		Cred:   auth.New("ak", "sk", ""),
		Region: region,
		clientOptions: []Option{
			WithHTTPClient(&http.Client{Transport: fn}),
			WithRetryPolicy(volcapi.RetryPolicy{MaxAttempts: 1}),
			WithClock(func() time.Time { return time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC) }),
		},
	}
}

func TestGetBucketACLCollapsesGrantsToCanned(t *testing.T) {
	t.Parallel()

	const body = `{"Owner":{"ID":"10001"},"Grants":[{"Grantee":{"Type":"Group","Canned":"AllUsers"},"Permission":"READ"}]}`

	driver := newTOSACLDriver(t, "cn-beijing", func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if got := req.URL.RawQuery; got != "acl=" {
			t.Fatalf("expected ?acl, got %q", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})
	client, err := driver.NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	got, err := client.GetBucketACL(context.Background(), "ctk-demo", "cn-beijing")
	if err != nil {
		t.Fatalf("GetBucketACL: %v", err)
	}
	if got != TOSACLPublicRead {
		t.Errorf("expected public-read, got %q", got)
	}
}

func TestPutBucketACLSendsHeaderAndQuery(t *testing.T) {
	t.Parallel()

	var sawACL string
	var sawQuery string
	driver := newTOSACLDriver(t, "cn-beijing", func(req *http.Request) (*http.Response, error) {
		sawACL = req.Header.Get("x-tos-acl")
		sawQuery = req.URL.RawQuery
		if req.Method != http.MethodPut {
			t.Fatalf("expected PUT, got %s", req.Method)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    req,
		}, nil
	})
	client, err := driver.NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if err := client.PutBucketACL(context.Background(), "ctk-demo", "cn-beijing", TOSACLPublicRead); err != nil {
		t.Fatalf("PutBucketACL: %v", err)
	}
	if sawACL != "public-read" {
		t.Errorf("expected x-tos-acl=public-read, got %q", sawACL)
	}
	if sawQuery != "acl=" {
		t.Errorf("expected ?acl, got %q", sawQuery)
	}
}

func TestExposeBucketDefaultsToPublicRead(t *testing.T) {
	t.Parallel()

	var sawACL string
	driver := newTOSACLDriver(t, "cn-beijing", func(req *http.Request) (*http.Response, error) {
		sawACL = req.Header.Get("x-tos-acl")
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    req,
		}, nil
	})
	applied, err := driver.ExposeBucket(context.Background(), "ctk-demo", "")
	if err != nil {
		t.Fatalf("ExposeBucket: %v", err)
	}
	if applied != TOSACLPublicRead || sawACL != "public-read" {
		t.Errorf("expected public-read, applied=%q, sent=%q", applied, sawACL)
	}
}

func TestNormalizeTOSACL(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":                    TOSACLPrivate,
		" private ":           TOSACLPrivate,
		"public-read":         TOSACLPublicRead,
		"PublicRead":          TOSACLPublicRead,
		"Blob":                TOSACLPublicRead,
		"public-read-write":   TOSACLPublicReadWrite,
		"writable":            TOSACLPublicReadWrite,
		"authenticated-read":  TOSACLAuthenticatedRead,
		"bucket-owner-read":   TOSACLBucketOwnerRead,
	}
	for in, want := range cases {
		if got := NormalizeTOSACL(in); got != want {
			t.Errorf("NormalizeTOSACL(%q) = %q, want %q", in, got, want)
		}
	}
}
