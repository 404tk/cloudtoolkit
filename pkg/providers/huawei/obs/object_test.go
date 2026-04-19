package obs

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
)

func TestClientListObjectsUsesBucketScopedEndpoint(t *testing.T) {
	ts := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	signed, err := Sign(&SignRequest{
		Method:    http.MethodGet,
		Path:      "/examplebucket",
		Scheme:    authSchemeV2,
		AccessKey: "AKIDEXAMPLE",
		SecretKey: "SECRETKEYEXAMPLE",
		Timestamp: ts,
	})
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	transport := &routingTransport{
		t: t,
		routes: map[string]routeResponse{
			"GET obs.cn-south-1.myhuaweicloud.com /examplebucket?max-keys=100": {
				body: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<ListBucketResult xmlns="http://obs.cn-south-1.myhuaweicloud.com/doc/2015-06-30/">
  <Name>examplebucket</Name>
  <MaxKeys>100</MaxKeys>
  <IsTruncated>false</IsTruncated>
  <Contents>
    <Key>object001</Key>
    <Size>12041</Size>
  </Contents>
</ListBucketResult>`,
			},
		},
		wantDate:          signed.Get(dateHeader),
		wantAuthorization: signed.Get(authHeader),
	}

	client := NewClient(
		auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", "cn-south-1", false),
		WithHTTPClient(&http.Client{Transport: transport}),
		WithRetryPolicy(noopRetryPolicy{}),
		WithClock(func() time.Time { return ts }),
	)

	resp, err := client.ListObjects(context.Background(), "examplebucket", "cn-south-1", "", 100)
	if err != nil {
		t.Fatalf("ListObjects() error = %v", err)
	}
	if resp.Name != "examplebucket" || len(resp.Objects) != 1 || resp.Objects[0].Key != "object001" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if strings.Join(transport.calls, ",") != "obs.cn-south-1.myhuaweicloud.com" {
		t.Fatalf("unexpected request order: %v", transport.calls)
	}
}

func TestDriverCountBucketObjectsFallsBackToLastObjectKey(t *testing.T) {
	ts := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	signed, err := Sign(&SignRequest{
		Method:    http.MethodGet,
		Path:      "/examplebucket",
		Scheme:    authSchemeV2,
		AccessKey: "AKIDEXAMPLE",
		SecretKey: "SECRETKEYEXAMPLE",
		Timestamp: ts,
	})
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	transport := &routingTransport{
		t: t,
		routes: map[string]routeResponse{
			"GET obs.cn-south-1.myhuaweicloud.com /examplebucket?max-keys=1000": {
				body: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<ListBucketResult>
  <Name>examplebucket</Name>
  <MaxKeys>1000</MaxKeys>
  <IsTruncated>true</IsTruncated>
  <Contents><Key>a.txt</Key><Size>1</Size></Contents>
  <Contents><Key>b.txt</Key><Size>2</Size></Contents>
</ListBucketResult>`,
			},
			"GET obs.cn-south-1.myhuaweicloud.com /examplebucket?marker=b.txt&max-keys=1000": {
				body: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<ListBucketResult>
  <Name>examplebucket</Name>
  <MaxKeys>1000</MaxKeys>
  <IsTruncated>false</IsTruncated>
  <Contents><Key>c.txt</Key><Size>3</Size></Contents>
</ListBucketResult>`,
			},
		},
		wantDate:          signed.Get(dateHeader),
		wantAuthorization: signed.Get(authHeader),
	}

	driver := &Driver{
		Cred: auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", "cn-south-1", false),
		Client: NewClient(
			auth.New("AKIDEXAMPLE", "SECRETKEYEXAMPLE", "cn-south-1", false),
			WithHTTPClient(&http.Client{Transport: transport}),
			WithRetryPolicy(noopRetryPolicy{}),
			WithClock(func() time.Time { return ts }),
		),
	}

	count, err := driver.countBucketObjects(context.Background(), "examplebucket", "cn-south-1", nil)
	if err != nil {
		t.Fatalf("countBucketObjects() error = %v", err)
	}
	if count != 3 {
		t.Fatalf("unexpected object count: %d", count)
	}
	if got, want := strings.Join(transport.calls, ","), "obs.cn-south-1.myhuaweicloud.com,obs.cn-south-1.myhuaweicloud.com"; got != want {
		t.Fatalf("unexpected request order: %s", got)
	}
}
