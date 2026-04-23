package oss

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	awsapi "github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/auth"
)

func TestDriverListBucketsUsesFixedRegionAndResolvesBucketRegions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/regions/cn-north-1/buckets":
			_, _ = w.Write([]byte(`{"requestId":"req-oss","result":{"buckets":[{"name":"bucket-a"},{"name":"bucket-b"}]}}`))
		case r.Method == http.MethodHead && r.URL.Path == "/v1/regions/cn-north-1/buckets/bucket-a":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"requestId":"req-miss-a","error":{"status":"404","code":404,"message":"bucket not found"}}`))
		case r.Method == http.MethodHead && r.URL.Path == "/v1/regions/cn-east-1/buckets/bucket-a":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodHead && r.URL.Path == "/v1/regions/cn-north-1/buckets/bucket-b":
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL)}
	got, err := driver.ListBuckets(context.Background())
	if err != nil {
		t.Fatalf("ListBuckets() error = %v", err)
	}
	if len(got) != 2 || got[0].BucketName != "bucket-a" || got[1].BucketName != "bucket-b" {
		t.Fatalf("unexpected buckets: %+v", got)
	}
	if got[0].Region != "cn-east-1" || got[1].Region != "cn-north-1" {
		t.Fatalf("unexpected bucket regions: %+v", got)
	}
}

func TestDriverResolveBucketRegionUsesExplicitRegionFirst(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/regions/cn-south-1/buckets/demo-bucket":
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	driver := &Driver{
		Client: newTestClient(server.URL),
		Region: "cn-south-1",
	}
	got, err := driver.ResolveBucketRegion(context.Background(), "demo-bucket")
	if err != nil {
		t.Fatalf("ResolveBucketRegion() error = %v", err)
	}
	if got != "cn-south-1" {
		t.Fatalf("ResolveBucketRegion() = %q, want cn-south-1", got)
	}
}

func TestClientListObjectsV2UsesServiceHostAndContinuationToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "s3.cn-north-1.jdcloud-oss.com" {
			t.Fatalf("unexpected host: %s", r.Host)
		}
		if r.URL.Path != "/demo-bucket" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("list-type"); got != "2" {
			t.Fatalf("unexpected list-type: %s", got)
		}
		if got := r.URL.Query().Get("continuation-token"); got != "page-2" {
			t.Fatalf("unexpected continuation-token: %s", got)
		}
		if got := r.URL.Query().Get("max-keys"); got != "100" {
			t.Fatalf("unexpected max-keys: %s", got)
		}
		if authz := r.Header.Get("Authorization"); !strings.HasPrefix(authz, "AWS4-HMAC-SHA256 Credential=AKID/20260419/cn-north-1/s3/aws4_request, SignedHeaders=") {
			t.Fatalf("unexpected authorization: %s", authz)
		}
		_, _ = w.Write([]byte(`<ListBucketResult><IsTruncated>true</IsTruncated><NextContinuationToken>page-3</NextContinuationToken><Contents><Key>logs/a.txt</Key><Size>12</Size></Contents></ListBucketResult>`))
	}))
	defer server.Close()

	client := NewClient(
		auth.New("AKID", "SECRET", ""),
		awsapi.WithHTTPClient(rewriteHostClient(server.URL)),
		awsapi.WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
		awsapi.WithRetryPolicy(awsapi.RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)

	got, err := client.ListObjectsV2(context.Background(), "demo-bucket", "cn-north-1", "page-2", 100)
	if err != nil {
		t.Fatalf("ListObjectsV2() error = %v", err)
	}
	if !got.IsTruncated || got.NextContinuationToken != "page-3" {
		t.Fatalf("unexpected pagination: %+v", got)
	}
	if len(got.Objects) != 1 || got.Objects[0].Key != "logs/a.txt" || got.Objects[0].Size != 12 {
		t.Fatalf("unexpected objects: %+v", got)
	}
}

func newTestClient(baseURL string) *api.Client {
	return api.NewClient(
		auth.New("AKID", "SECRET", ""),
		api.WithBaseURL(baseURL),
		api.WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
		api.WithNonceFunc(func() string { return "3233e986-8ad0-41b6-a9f7-83b052dc5577" }),
		api.WithRetryPolicy(api.RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)
}

func rewriteHostClient(rawURL string) *http.Client {
	target, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return &http.Client{
		Transport: &rewriteHostTransport{
			target: target,
			base: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

type rewriteHostTransport struct {
	target *url.URL
	base   http.RoundTripper
}

func (t *rewriteHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	urlCopy := *clone.URL
	urlCopy.Scheme = t.target.Scheme
	urlCopy.Host = t.target.Host
	clone.URL = &urlCopy
	return t.base.RoundTrip(clone)
}
