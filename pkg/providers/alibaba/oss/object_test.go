package oss

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
)

func TestClientListObjectsV2UsesBucketScopedEndpoint(t *testing.T) {
	client := NewClient(
		aliauth.New("ak", "sk", "sts-token"),
		WithHTTPClient(&http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodGet {
					t.Fatalf("unexpected method: %s", req.Method)
				}
				if req.URL.Host != "examplebucket.oss-cn-shanghai.aliyuncs.com" {
					t.Fatalf("unexpected host: %s", req.URL.Host)
				}
				if req.URL.Path != "/" {
					t.Fatalf("unexpected path: %s", req.URL.Path)
				}
				if got := req.URL.Query().Get("list-type"); got != "2" {
					t.Fatalf("unexpected list-type: %s", got)
				}
				if got := req.URL.Query().Get("encoding-type"); got != "url" {
					t.Fatalf("unexpected encoding-type: %s", got)
				}
				if got := req.URL.Query().Get("max-keys"); got != "100" {
					t.Fatalf("unexpected max-keys: %s", got)
				}
				if got := req.Header.Get("X-Oss-Security-Token"); got != "sts-token" {
					t.Fatalf("unexpected security token header: %s", got)
				}
				if got := req.Header.Get("Authorization"); got == "" || !strings.HasPrefix(got, "OSS ak:") {
					t.Fatalf("unexpected authorization: %s", got)
				}
				body := `<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult>
  <Name>examplebucket</Name>
  <MaxKeys>100</MaxKeys>
  <IsTruncated>false</IsTruncated>
  <Contents>
    <Key>dir%2Falpha.txt</Key>
    <Size>12</Size>
  </Contents>
</ListBucketResult>`
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(body)),
					Request:    req,
				}, nil
			}),
		}),
		WithRetryPolicy(api.RetryPolicy{MaxAttempts: 1}),
		WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
	)

	resp, err := client.ListObjectsV2(context.Background(), "examplebucket", "cn-shanghai", "", 100)
	if err != nil {
		t.Fatalf("ListObjectsV2() error = %v", err)
	}
	if resp.Name != "examplebucket" || len(resp.Objects) != 1 || resp.Objects[0].Key != "dir/alpha.txt" || resp.Objects[0].Size != 12 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestDriverCountBucketObjectsPaginatesContinuationToken(t *testing.T) {
	driver := &Driver{
		Cred: aliauth.New("ak", "sk", ""),
		Client: NewClient(
			aliauth.New("ak", "sk", ""),
			WithHTTPClient(&http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					if req.URL.Host != "examplebucket.oss-cn-shanghai.aliyuncs.com" {
						t.Fatalf("unexpected host: %s", req.URL.Host)
					}

					query, err := url.ParseQuery(req.URL.RawQuery)
					if err != nil {
						t.Fatalf("ParseQuery() error = %v", err)
					}
					if got := query.Get("list-type"); got != "2" {
						t.Fatalf("unexpected list-type: %s", got)
					}
					if got := query.Get("encoding-type"); got != "url" {
						t.Fatalf("unexpected encoding-type: %s", got)
					}
					if got := query.Get("max-keys"); got != "1000" {
						t.Fatalf("unexpected max-keys: %s", got)
					}

					switch query.Get("continuation-token") {
					case "":
						return &http.Response{
							StatusCode: http.StatusOK,
							Header:     make(http.Header),
							Body: io.NopCloser(strings.NewReader(`<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult>
  <Name>examplebucket</Name>
  <MaxKeys>1000</MaxKeys>
  <IsTruncated>true</IsTruncated>
  <NextContinuationToken>page-2</NextContinuationToken>
  <Contents><Key>a.txt</Key><Size>1</Size></Contents>
  <Contents><Key>b.txt</Key><Size>2</Size></Contents>
</ListBucketResult>`)),
							Request: req,
						}, nil
					case "page-2":
						return &http.Response{
							StatusCode: http.StatusOK,
							Header:     make(http.Header),
							Body: io.NopCloser(strings.NewReader(`<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult>
  <Name>examplebucket</Name>
  <MaxKeys>1000</MaxKeys>
  <IsTruncated>false</IsTruncated>
  <Contents><Key>c.txt</Key><Size>3</Size></Contents>
</ListBucketResult>`)),
							Request: req,
						}, nil
					default:
						t.Fatalf("unexpected continuation-token: %s", query.Get("continuation-token"))
						return nil, nil
					}
				}),
			}),
			WithRetryPolicy(api.RetryPolicy{MaxAttempts: 1}),
			WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
		),
	}

	count, err := driver.countBucketObjects(context.Background(), "examplebucket", "cn-shanghai", nil)
	if err != nil {
		t.Fatalf("countBucketObjects() error = %v", err)
	}
	if count != 3 {
		t.Fatalf("unexpected object count: %d", count)
	}
}
