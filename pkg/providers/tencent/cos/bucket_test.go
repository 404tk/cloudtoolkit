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

func TestDriverGetBucketsMapsServiceResponse(t *testing.T) {
	client := NewClient(
		auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", "TOKENEXAMPLE"),
		WithHTTPClient(&http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodGet {
					t.Fatalf("unexpected method: %s", r.Method)
				}
				if r.URL.Host != "service.cos.myqcloud.com" {
					t.Fatalf("unexpected host: %s", r.URL.Host)
				}
				if r.URL.Path != "/" {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
				if r.Header.Get("x-cos-security-token") != "TOKENEXAMPLE" {
					t.Fatalf("unexpected token header: %q", r.Header.Get("x-cos-security-token"))
				}
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" {
					t.Fatal("missing authorization header")
				}
				if !strings.Contains(authHeader, "q-header-list=host;x-cos-security-token") {
					t.Fatalf("unexpected authorization header: %s", authHeader)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`<ListAllMyBucketsResult>
	<Owner>
		<ID>xbaccxx</ID>
		<DisplayName>100000760461</DisplayName>
	</Owner>
	<Buckets>
		<Bucket>
			<Name>huadong-1253846586</Name>
			<Location>ap-shanghai</Location>
			<CreationDate>2017-06-16T13:08:28Z</CreationDate>
		</Bucket>
		<Bucket>
			<Name>huanan-1253846586</Name>
			<Location>ap-guangzhou</Location>
			<CreationDate>2017-06-10T09:00:07Z</CreationDate>
		</Bucket>
	</Buckets>
</ListAllMyBucketsResult>`)),
					Request: r,
				}, nil
			}),
		}),
		WithRetryPolicy(api.RetryPolicy{MaxAttempts: 1}),
		WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
	)

	driver := &Driver{
		Credential: auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", "TOKENEXAMPLE"),
		Client:     client,
	}

	got, err := driver.GetBuckets(context.Background())
	if err != nil {
		t.Fatalf("GetBuckets() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("unexpected bucket count: %d", len(got))
	}
	if got[0].BucketName != "huadong-1253846586" || got[0].Region != "ap-shanghai" {
		t.Fatalf("unexpected first bucket: %+v", got[0])
	}
	if got[1].BucketName != "huanan-1253846586" || got[1].Region != "ap-guangzhou" {
		t.Fatalf("unexpected second bucket: %+v", got[1])
	}
}

func TestDriverGetBucketsReturnsServiceError(t *testing.T) {
	client := NewClient(
		auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", ""),
		WithHTTPClient(&http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusForbidden,
					Header: http.Header{
						"X-Cos-Request-Id": []string{"req-1"},
					},
					Body:    io.NopCloser(strings.NewReader(`<Error><Code>AccessDenied</Code><Message>denied</Message></Error>`)),
					Request: r,
				}, nil
			}),
		}),
		WithRetryPolicy(api.RetryPolicy{MaxAttempts: 1}),
		WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
	)

	driver := &Driver{
		Credential: auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", ""),
		Client:     client,
	}

	_, err := driver.GetBuckets(context.Background())
	if err == nil {
		t.Fatal("expected service error")
	}
	if !strings.Contains(err.Error(), "AccessDenied") || !strings.Contains(err.Error(), "request_id=req-1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}
