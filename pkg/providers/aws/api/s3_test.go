package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/auth"
)

func TestS3ListBucketsParsesBucketsAndRegionHints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("X-Amz-Content-Sha256"); got != hashSHA256Hex(nil) {
			t.Fatalf("unexpected payload hash header: %s", got)
		}
		authz := r.Header.Get("Authorization")
		if !strings.Contains(authz, "/us-east-1/s3/aws4_request") {
			t.Fatalf("unexpected authorization header: %s", authz)
		}
		_, _ = w.Write([]byte(`
<ListAllMyBucketsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Buckets>
    <Bucket>
      <Name>alpha</Name>
      <BucketRegion>ap-southeast-1</BucketRegion>
    </Bucket>
    <Bucket>
      <Name>beta</Name>
      <BucketRegion>EU</BucketRegion>
    </Bucket>
  </Buckets>
</ListAllMyBucketsResult>`))
	}))
	defer server.Close()

	client := newS3TestClient(server.URL)
	got, err := client.ListBuckets(context.Background(), "us-east-1")
	if err != nil {
		t.Fatalf("ListBuckets() error = %v", err)
	}
	if len(got.Buckets) != 2 {
		t.Fatalf("unexpected bucket count: %d", len(got.Buckets))
	}
	if got.Buckets[0].Name != "alpha" || got.Buckets[0].BucketRegion != "ap-southeast-1" {
		t.Fatalf("unexpected first bucket: %+v", got.Buckets[0])
	}
	if got.Buckets[1].Name != "beta" || got.Buckets[1].BucketRegion != "eu-west-1" {
		t.Fatalf("unexpected second bucket: %+v", got.Buckets[1])
	}
}

func TestS3GetBucketLocationParsesLegacyEU(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/demo-bucket" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if _, ok := r.URL.Query()["location"]; !ok {
			t.Fatalf("missing location query: %s", r.URL.RawQuery)
		}
		authz := r.Header.Get("Authorization")
		if !strings.Contains(authz, "/us-west-2/s3/aws4_request") {
			t.Fatalf("unexpected authorization header: %s", authz)
		}
		_, _ = w.Write([]byte(`<LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/">EU</LocationConstraint>`))
	}))
	defer server.Close()

	client := newS3TestClient(server.URL)
	got, err := client.GetBucketLocation(context.Background(), "us-west-2", "demo-bucket")
	if err != nil {
		t.Fatalf("GetBucketLocation() error = %v", err)
	}
	if got.Region != "eu-west-1" {
		t.Fatalf("unexpected region: %+v", got)
	}
}

func TestS3ListObjectsV2ParsesContentsAndNextToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/demo-bucket" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("list-type"); got != "2" {
			t.Fatalf("unexpected list-type: %s", got)
		}
		if got := r.URL.Query().Get("continuation-token"); got != "page-2" {
			t.Fatalf("unexpected continuation token: %s", got)
		}
		if got := r.URL.Query().Get("max-keys"); got != "1000" {
			t.Fatalf("unexpected max-keys: %s", got)
		}
		authz := r.Header.Get("Authorization")
		if !strings.Contains(authz, "/ap-southeast-1/s3/aws4_request") {
			t.Fatalf("unexpected authorization header: %s", authz)
		}
		_, _ = w.Write([]byte(`
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <IsTruncated>true</IsTruncated>
  <NextContinuationToken>page-3</NextContinuationToken>
  <Contents>
    <Key>alpha.txt</Key>
    <Size>12</Size>
  </Contents>
  <Contents>
    <Key>logs/app.log</Key>
    <Size>34</Size>
  </Contents>
</ListBucketResult>`))
	}))
	defer server.Close()

	client := newS3TestClient(server.URL)
	got, err := client.ListObjectsV2(context.Background(), "ap-southeast-1", "demo-bucket", "page-2", 1000)
	if err != nil {
		t.Fatalf("ListObjectsV2() error = %v", err)
	}
	if !got.IsTruncated || got.NextContinuationToken != "page-3" {
		t.Fatalf("unexpected page metadata: %+v", got)
	}
	if len(got.Objects) != 2 || got.Objects[1].Key != "logs/app.log" || got.Objects[1].Size != 34 {
		t.Fatalf("unexpected objects: %+v", got.Objects)
	}
}

func TestS3ErrorResponseDecodesToAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`
<Error>
  <Code>NoSuchBucket</Code>
  <Message>The specified bucket does not exist.</Message>
  <RequestId>req-s3-error</RequestId>
</Error>`))
	}))
	defer server.Close()

	client := newS3TestClient(server.URL)
	_, err := client.GetBucketLocation(context.Background(), "us-east-1", "missing-bucket")
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Code != "NoSuchBucket" || apiErr.RequestID != "req-s3-error" {
		t.Fatalf("unexpected api error: %+v", apiErr)
	}
}

func newS3TestClient(baseURL string) *Client {
	return NewClient(
		auth.New("AKID", "SECRET", ""),
		WithBaseURL(baseURL),
		WithClock(func() time.Time { return time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC) }),
		WithRetryPolicy(RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)
}
