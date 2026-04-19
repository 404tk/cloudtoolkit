package tencent

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/cos"
)

func TestBucketInfosResolvesRegionFromBucketListWhenProviderRegionIsAll(t *testing.T) {
	driver := &cos.Driver{
		Credential: auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", ""),
		Client: cos.NewClient(
			auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", ""),
			cos.WithHTTPClient(&http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					if r.URL.Host != "service.cos.myqcloud.com" {
						t.Fatalf("unexpected host: %s", r.URL.Host)
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     make(http.Header),
						Body: io.NopCloser(strings.NewReader(`<ListAllMyBucketsResult>
	<Buckets>
		<Bucket><Name>a-1250000000</Name><Location>ap-guangzhou</Location></Bucket>
		<Bucket><Name>b-1250000000</Name><Location>ap-shanghai</Location></Bucket>
	</Buckets>
</ListAllMyBucketsResult>`)),
						Request: r,
					}, nil
				}),
			}),
			cos.WithRetryPolicy(api.RetryPolicy{MaxAttempts: 1}),
			cos.WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
		),
	}

	provider := &Provider{region: "all"}
	infos, err := provider.bucketInfos(context.Background(), driver, "b-1250000000")
	if err != nil {
		t.Fatalf("bucketInfos() error = %v", err)
	}
	if len(infos) != 1 || infos["b-1250000000"] != "ap-shanghai" {
		t.Fatalf("unexpected bucket infos: %+v", infos)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}
