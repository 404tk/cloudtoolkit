package alibaba

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/oss"
)

func TestBucketInfosResolvesRegionFromBucketListWhenProviderRegionIsAll(t *testing.T) {
	driver := &oss.Driver{
		Cred: aliauth.New("ak", "sk", ""),
		Client: oss.NewClient(
			aliauth.New("ak", "sk", ""),
			oss.WithHTTPClient(&http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					if req.URL.Host != "oss-cn-hangzhou.aliyuncs.com" {
						t.Fatalf("unexpected host: %s", req.URL.Host)
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     make(http.Header),
						Body: io.NopCloser(strings.NewReader(`<?xml version="1.0" encoding="UTF-8"?>
<ListAllMyBucketsResult>
  <Buckets>
    <Bucket><Name>a</Name><Location>oss-cn-hangzhou</Location></Bucket>
    <Bucket><Name>b</Name><Location>oss-cn-shanghai</Location></Bucket>
  </Buckets>
</ListAllMyBucketsResult>`)),
						Request: req,
					}, nil
				}),
			}),
			oss.WithRetryPolicy(api.RetryPolicy{MaxAttempts: 1}),
			oss.WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
		),
	}

	provider := &Provider{region: "all"}
	infos, err := provider.bucketInfos(context.Background(), driver, "b")
	if err != nil {
		t.Fatalf("bucketInfos() error = %v", err)
	}
	if len(infos) != 1 || infos["b"] != "cn-shanghai" {
		t.Fatalf("unexpected bucket infos: %+v", infos)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
