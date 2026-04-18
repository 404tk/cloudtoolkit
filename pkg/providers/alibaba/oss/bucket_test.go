package oss

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
	aliyunoss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

func TestNewClientUsesNormalizedEndpointAndToken(t *testing.T) {
	t.Parallel()

	driver := Driver{
		Cred:   aliauth.New("ak", "sk", "sts-token"),
		Region: "all",
	}

	client, err := driver.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client.Config.Endpoint != "https://oss-cn-hangzhou.aliyuncs.com" {
		t.Fatalf("unexpected endpoint: %s", client.Config.Endpoint)
	}
	if client.Config.SecurityToken != "sts-token" {
		t.Fatalf("unexpected security token: %s", client.Config.SecurityToken)
	}
}

func TestNewClientValidatesCredential(t *testing.T) {
	t.Parallel()

	driver := Driver{
		Cred:   aliauth.New("", "sk", ""),
		Region: "cn-hangzhou",
	}

	if _, err := driver.NewClient(); err == nil {
		t.Fatal("NewClient() error = nil, want validation failure")
	}
}

func TestGetBucketsMapsResponseAndUsesToken(t *testing.T) {
	t.Parallel()

	driver := Driver{
		Cred:   aliauth.New("ak", "sk", "sts-token"),
		Region: "cn-shanghai",
		clientOptions: []aliyunoss.ClientOption{
			aliyunoss.HTTPClient(&http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					if req.Method != http.MethodGet {
						t.Fatalf("unexpected method: %s", req.Method)
					}
					if req.URL.Host != "oss-cn-shanghai.aliyuncs.com" {
						t.Fatalf("unexpected host: %s", req.URL.Host)
					}
					if got := req.Header.Get("X-Oss-Security-Token"); got != "sts-token" {
						t.Fatalf("unexpected security token header: %s", got)
					}
					body := `<?xml version="1.0" encoding="UTF-8"?>
<ListAllMyBucketsResult>
  <Buckets>
    <Bucket>
      <Name>bucket-a</Name>
      <Location>oss-cn-hangzhou</Location>
    </Bucket>
    <Bucket>
      <Name>bucket-b</Name>
      <Location>oss-cn-shanghai</Location>
    </Bucket>
  </Buckets>
</ListAllMyBucketsResult>`
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     http.Header{"Content-Type": []string{"application/xml"}},
						Body:       io.NopCloser(strings.NewReader(body)),
					}, nil
				}),
			}),
		},
	}

	got, err := driver.GetBuckets(context.Background())
	if err != nil {
		t.Fatalf("GetBuckets() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("unexpected bucket count: %d", len(got))
	}
	if got[0].BucketName != "bucket-a" || got[0].Region != "cn-hangzhou" {
		t.Fatalf("unexpected first bucket: %+v", got[0])
	}
	if got[1].BucketName != "bucket-b" || got[1].Region != "cn-shanghai" {
		t.Fatalf("unexpected second bucket: %+v", got[1])
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
