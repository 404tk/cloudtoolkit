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

func TestDriverGetBucketsMapsResponseAndSignsRequest(t *testing.T) {
	expectedTime := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	client := NewClient(
		auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", "TOKENEXAMPLE"),
		WithHTTPClient(&http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodGet {
					t.Fatalf("unexpected method: %s", req.Method)
				}
				if req.URL.Host != "tos-cn-beijing.volces.com" {
					t.Fatalf("unexpected url host: %s", req.URL.Host)
				}
				if req.Host != "tos-cn-beijing.volces.com" {
					t.Fatalf("unexpected host header: %s", req.Host)
				}
				if req.URL.Path != "/" {
					t.Fatalf("unexpected path: %s", req.URL.Path)
				}
				if got := req.Header.Get(headerDate); got != expectedTime.Format(http.TimeFormat) {
					t.Fatalf("unexpected date header: %s", got)
				}
				if got := req.Header.Get(headerXDate); got != "20260419T120000Z" {
					t.Fatalf("unexpected x-tos-date: %s", got)
				}
				if got := req.Header.Get(headerSecurityToken); got != "TOKENEXAMPLE" {
					t.Fatalf("unexpected security token: %s", got)
				}
				authz := req.Header.Get(headerAuthorization)
				if !strings.Contains(authz, "TOS4-HMAC-SHA256 Credential=AKIDEXAMPLE/20260419/cn-beijing/tos/request") {
					t.Fatalf("unexpected authorization: %s", authz)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`{
  "Buckets": [
    {"Name":"bucket-a","Location":"cn-beijing"},
    {"Name":"bucket-b","Location":"cn-guangzhou"}
  ],
  "Owner": {"ID":"2000000001"}
}`)),
					Request: req,
				}, nil
			}),
		}),
		WithRetryPolicy(volcapi.RetryPolicy{MaxAttempts: 1}),
		WithClock(func() time.Time { return expectedTime }),
	)

	driver := &Driver{
		Cred:   auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", "TOKENEXAMPLE"),
		Region: "cn-beijing",
		Client: client,
	}

	got, err := driver.GetBuckets(context.Background())
	if err != nil {
		t.Fatalf("GetBuckets() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("unexpected bucket count: %d", len(got))
	}
	if got[0].BucketName != "bucket-a" || got[0].Region != "cn-beijing" {
		t.Fatalf("unexpected first bucket: %+v", got[0])
	}
	if got[1].BucketName != "bucket-b" || got[1].Region != "cn-guangzhou" {
		t.Fatalf("unexpected second bucket: %+v", got[1])
	}
}

func TestClientListObjectsV2UsesBucketHostAndContinuationToken(t *testing.T) {
	client := NewClient(
		auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", ""),
		WithHTTPClient(&http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodGet {
					t.Fatalf("unexpected method: %s", req.Method)
				}
				if req.URL.Host != "demo-bucket.tos-cn-guangzhou.volces.com" {
					t.Fatalf("unexpected url host: %s", req.URL.Host)
				}
				if req.Host != "demo-bucket.tos-cn-guangzhou.volces.com" {
					t.Fatalf("unexpected request host: %s", req.Host)
				}
				if got := req.URL.RawQuery; got != "continuation-token=next-token&list-type=2&max-keys=100" {
					t.Fatalf("unexpected query: %s", got)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`{
  "Name":"demo-bucket",
  "IsTruncated": true,
  "NextContinuationToken":"token-2",
  "Contents": [
    {"Key":"a.txt","Size":12},
    {"Key":"b.txt","Size":34}
  ]
}`)),
					Request: req,
				}, nil
			}),
		}),
		WithRetryPolicy(volcapi.RetryPolicy{MaxAttempts: 1}),
	)

	resp, err := client.ListObjectsV2(context.Background(), "demo-bucket", "cn-guangzhou", "next-token", 100)
	if err != nil {
		t.Fatalf("ListObjectsV2() error = %v", err)
	}
	if !resp.IsTruncated || resp.NextContinuationToken != "token-2" {
		t.Fatalf("unexpected pagination response: %+v", resp)
	}
	if len(resp.Contents) != 2 || resp.Contents[0].Key != "a.txt" || resp.Contents[1].Size != 34 {
		t.Fatalf("unexpected contents: %+v", resp.Contents)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
