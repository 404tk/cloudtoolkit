package obs

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
)

func TestSignListBucketsRequest(t *testing.T) {
	ts := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	got, err := Sign(&SignRequest{
		Method:    http.MethodGet,
		Path:      "/",
		AccessKey: "AKIDEXAMPLE",
		SecretKey: "SECRETKEYEXAMPLE",
		Timestamp: ts,
	})
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	const wantDate = "Sun, 19 Apr 2026 12:00:00 GMT"
	const wantStringToSign = "GET\n\n\nSun, 19 Apr 2026 12:00:00 GMT\n/"
	const wantAuthorization = "OBS AKIDEXAMPLE:TaTgztIT4wx8Sq3AlVjJljVLslY="

	if got.Get(dateHeader) != wantDate {
		t.Fatalf("unexpected date header: %q", got.Get(dateHeader))
	}
	if got.Get(authHeader) != wantAuthorization {
		t.Fatalf("unexpected authorization header:\n got:  %s\nwant: %s", got.Get(authHeader), wantAuthorization)
	}
	if stringToSign := buildStringToSign(http.MethodGet, "/", nil, got); stringToSign != wantStringToSign {
		t.Fatalf("unexpected string to sign:\n got:  %q\nwant: %q", stringToSign, wantStringToSign)
	}
}

func TestDriverGetBucketsUsesSingleEndpointAndMapsResponseLocations(t *testing.T) {
	transport := &routingTransport{
		t: t,
		routes: map[string]routeResponse{
			"GET obs.cn-north-4.myhuaweicloud.com /?": {
				body: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<ListAllMyBucketsResult xmlns="http://obs.cn-north-4.myhuaweicloud.com/doc/2015-06-30/">
  <Owner><ID>783fc6652cf246c096ea836694f71855</ID></Owner>
  <Buckets>
    <Bucket>
      <Name>examplebucket01</Name>
      <CreationDate>2018-06-21T09:15:01.032Z</CreationDate>
      <Location>cn-north-4</Location>
      <BucketType>OBJECT</BucketType>
    </Bucket>
    <Bucket>
      <Name>examplebucket02</Name>
      <CreationDate>2018-06-22T03:56:33.700Z</CreationDate>
      <BucketType>OBJECT</BucketType>
    </Bucket>
  </Buckets>
</ListAllMyBucketsResult>`,
			},
		},
		wantDate:          "Sun, 19 Apr 2026 12:00:00 GMT",
		wantAuthorization: "OBS AKIDEXAMPLE:TaTgztIT4wx8Sq3AlVjJljVLslY=",
	}

	driver := &Driver{
		Cred:    auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", "cn-north-4", false),
		Regions: []string{"cn-east-201", "cn-east-3"},
		Client: NewClient(
			auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", "cn-north-4", false),
			WithHTTPClient(&http.Client{Transport: transport}),
			WithRetryPolicy(noopRetryPolicy{}),
			WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
		),
	}

	got, err := driver.GetBuckets(context.Background())
	if err != nil {
		t.Fatalf("GetBuckets() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("unexpected bucket count: %d", len(got))
	}
	if got[0].BucketName != "examplebucket01" || got[0].Region != "cn-north-4" {
		t.Fatalf("unexpected first bucket: %+v", got[0])
	}
	if got[1].BucketName != "examplebucket02" || got[1].Region != "cn-north-4" {
		t.Fatalf("unexpected second bucket: %+v", got[1])
	}
	if strings.Join(transport.calls, ",") != "obs.cn-north-4.myhuaweicloud.com" {
		t.Fatalf("unexpected request order: %v", transport.calls)
	}
}

func TestDriverGetBucketsPrefersExplicitRegionEndpoint(t *testing.T) {
	transport := &routingTransport{
		t: t,
		routes: map[string]routeResponse{
			"GET obs.cn-south-1.myhuaweicloud.com /?": {
				body: `<ListAllMyBucketsResult><Buckets></Buckets></ListAllMyBucketsResult>`,
			},
		},
		wantDate:          "Sun, 19 Apr 2026 12:00:00 GMT",
		wantAuthorization: "OBS AKIDEXAMPLE:TaTgztIT4wx8Sq3AlVjJljVLslY=",
	}

	driver := &Driver{
		Cred:    auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", "cn-south-1", false),
		Regions: []string{"cn-north-4", "cn-east-3"},
		Client: NewClient(
			auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", "cn-south-1", false),
			WithHTTPClient(&http.Client{Transport: transport}),
			WithRetryPolicy(noopRetryPolicy{}),
			WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
		),
	}

	got, err := driver.GetBuckets(context.Background())
	if err != nil {
		t.Fatalf("GetBuckets() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("unexpected bucket count: %d", len(got))
	}
	if strings.Join(transport.calls, ",") != "obs.cn-south-1.myhuaweicloud.com" {
		t.Fatalf("unexpected request order: %v", transport.calls)
	}
}

func TestDriverGetBucketsReturnsOBSServiceError(t *testing.T) {
	transport := &routingTransport{
		t: t,
		routes: map[string]routeResponse{
			"GET obs.cn-north-4.myhuaweicloud.com /?": {
				statusCode: http.StatusForbidden,
				headers: http.Header{
					"X-Obs-Request-Id": []string{"req-1"},
				},
				body: `<Error><Code>AccessDenied</Code><Message>denied</Message></Error>`,
			},
		},
		wantDate:          "Sun, 19 Apr 2026 12:00:00 GMT",
		wantAuthorization: "OBS AKIDEXAMPLE:TaTgztIT4wx8Sq3AlVjJljVLslY=",
	}

	driver := &Driver{
		Cred: auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", "cn-north-4", false),
		Client: NewClient(
			auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", "cn-north-4", false),
			WithHTTPClient(&http.Client{Transport: transport}),
			WithRetryPolicy(noopRetryPolicy{}),
			WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
		),
	}

	_, err := driver.GetBuckets(context.Background())
	if err == nil {
		t.Fatal("expected service error")
	}
	if !strings.Contains(err.Error(), "AccessDenied") || !strings.Contains(err.Error(), "request_id=req-1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDriverGetBucketsFallsBackToDefaultRegionWhenUnset(t *testing.T) {
	transport := &routingTransport{
		t: t,
		routes: map[string]routeResponse{
			"GET obs.cn-north-4.myhuaweicloud.com /?": {
				body: `<ListAllMyBucketsResult><Buckets></Buckets></ListAllMyBucketsResult>`,
			},
		},
		wantDate:          "Sun, 19 Apr 2026 12:00:00 GMT",
		wantAuthorization: "OBS AKIDEXAMPLE:TaTgztIT4wx8Sq3AlVjJljVLslY=",
	}

	driver := &Driver{
		Cred:    auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", "", false),
		Regions: nil,
		Client: NewClient(
			auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", "", false),
			WithHTTPClient(&http.Client{Transport: transport}),
			WithRetryPolicy(noopRetryPolicy{}),
			WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
		),
	}

	got, err := driver.GetBuckets(context.Background())
	if err != nil {
		t.Fatalf("GetBuckets() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("unexpected bucket count: %d", len(got))
	}
	if strings.Join(transport.calls, ",") != "obs.cn-north-4.myhuaweicloud.com" {
		t.Fatalf("unexpected request order: %v", transport.calls)
	}
}

type routeResponse struct {
	statusCode int
	headers    http.Header
	body       string
	err        error
}

type routingTransport struct {
	t                 *testing.T
	routes            map[string]routeResponse
	calls             []string
	wantDate          string
	wantAuthorization string
}

func (rt *routingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	rt.t.Helper()
	if got := r.Header.Get(dateHeader); got != rt.wantDate {
		rt.t.Fatalf("unexpected date header: %q", got)
	}
	if got := r.Header.Get(authHeader); got != rt.wantAuthorization {
		rt.t.Fatalf("unexpected authorization header: %q", got)
	}
	rt.calls = append(rt.calls, r.URL.Host)

	key := r.Method + " " + r.URL.Host + " " + r.URL.Path + "?" + r.URL.RawQuery
	resp, ok := rt.routes[key]
	if !ok {
		rt.t.Fatalf("unexpected request: %s", key)
	}
	if resp.err != nil {
		return nil, resp.err
	}
	statusCode := resp.statusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	headers := resp.headers.Clone()
	if headers == nil {
		headers = make(http.Header)
	}
	return &http.Response{
		StatusCode: statusCode,
		Header:     headers,
		Body:       io.NopCloser(strings.NewReader(resp.body)),
		Request:    r,
	}, nil
}

type noopRetryPolicy struct{}

func (noopRetryPolicy) Do(ctx context.Context, _ bool, fn func() (*http.Response, error)) (*http.Response, error) {
	return fn()
}

func TestDecodeErrorFallsBackToBody(t *testing.T) {
	err := decodeError(http.StatusBadGateway, http.Header{"X-Obs-Request-Id": []string{"req-2"}}, []byte("upstream failed"))
	if err == nil {
		t.Fatal("expected fallback error")
	}
	if !strings.Contains(err.Error(), "status=502") || !strings.Contains(err.Error(), "request_id=req-2") {
		t.Fatalf("unexpected fallback error: %v", err)
	}
}
