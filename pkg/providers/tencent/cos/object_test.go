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

func TestClientListObjectsUsesBucketScopedEndpoint(t *testing.T) {
	client := NewClient(
		auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", "TOKENEXAMPLE"),
		WithHTTPClient(&http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodGet {
					t.Fatalf("unexpected method: %s", r.Method)
				}
				if r.URL.Host != "examplebucket-1250000000.cos.ap-guangzhou.myqcloud.com" {
					t.Fatalf("unexpected host: %s", r.URL.Host)
				}
				if r.URL.Path != "/" {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
				if got := r.URL.Query().Get("max-keys"); got != "100" {
					t.Fatalf("unexpected max-keys: %s", got)
				}
				if got := r.URL.Query().Get("marker"); got != "" {
					t.Fatalf("unexpected marker: %s", got)
				}
				if r.Header.Get("x-cos-security-token") != "TOKENEXAMPLE" {
					t.Fatalf("unexpected token header: %q", r.Header.Get("x-cos-security-token"))
				}
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" {
					t.Fatal("missing authorization header")
				}
				if !strings.Contains(authHeader, "q-url-param-list=max-keys") {
					t.Fatalf("unexpected authorization header: %s", authHeader)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Name>examplebucket-1250000000</Name>
  <MaxKeys>100</MaxKeys>
  <IsTruncated>false</IsTruncated>
  <Contents>
    <Key>alpha.txt</Key>
    <Size>12</Size>
  </Contents>
</ListBucketResult>`)),
					Request: r,
				}, nil
			}),
		}),
		WithRetryPolicy(api.RetryPolicy{MaxAttempts: 1}),
		WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
	)

	resp, err := client.ListObjects(context.Background(), "examplebucket-1250000000", "ap-guangzhou", "", 100)
	if err != nil {
		t.Fatalf("ListObjects() error = %v", err)
	}
	if resp.Name != "examplebucket-1250000000" || len(resp.Objects) != 1 || resp.Objects[0].Key != "alpha.txt" || resp.Objects[0].Size != 12 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestDriverCountBucketObjectsPaginatesMarker(t *testing.T) {
	client := NewClient(
		auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", ""),
		WithHTTPClient(&http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.Host != "examplebucket-1250000000.cos.ap-guangzhou.myqcloud.com" {
					t.Fatalf("unexpected host: %s", r.URL.Host)
				}
				switch r.URL.RawQuery {
				case "max-keys=1000":
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     make(http.Header),
						Body: io.NopCloser(strings.NewReader(`
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Name>examplebucket-1250000000</Name>
  <MaxKeys>1000</MaxKeys>
  <IsTruncated>true</IsTruncated>
  <Contents><Key>a.txt</Key><Size>1</Size></Contents>
  <Contents><Key>b.txt</Key><Size>2</Size></Contents>
</ListBucketResult>`)),
						Request: r,
					}, nil
				case "marker=b.txt&max-keys=1000", "max-keys=1000&marker=b.txt":
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     make(http.Header),
						Body: io.NopCloser(strings.NewReader(`
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Name>examplebucket-1250000000</Name>
  <MaxKeys>1000</MaxKeys>
  <IsTruncated>false</IsTruncated>
  <Contents><Key>c.txt</Key><Size>3</Size></Contents>
</ListBucketResult>`)),
						Request: r,
					}, nil
				default:
					t.Fatalf("unexpected query: %s", r.URL.RawQuery)
					return nil, nil
				}
			}),
		}),
		WithRetryPolicy(api.RetryPolicy{MaxAttempts: 1}),
		WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
	)

	driver := &Driver{
		Credential: auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", ""),
		Client:     client,
	}

	count, err := driver.countBucketObjects(context.Background(), "examplebucket-1250000000", "ap-guangzhou", nil)
	if err != nil {
		t.Fatalf("countBucketObjects() error = %v", err)
	}
	if count != 3 {
		t.Fatalf("unexpected object count: %d", count)
	}
}
