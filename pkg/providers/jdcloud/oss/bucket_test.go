package oss

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/auth"
)

func TestDriverListBucketsUsesFixedRegion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/regions/cn-north-1/buckets" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"requestId":"req-oss","result":{"buckets":[{"name":"bucket-a"},{"name":"bucket-b"}]}}`))
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
