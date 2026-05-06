package dns

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

type noopRetryPolicy struct{}

func (noopRetryPolicy) Do(ctx context.Context, _ bool, fn func() (*http.Response, error)) (*http.Response, error) {
	return fn()
}

func newTestClient(t *testing.T, fn roundTripFunc) *api.Client {
	t.Helper()
	return api.NewClient(
		auth.New("AKID", "SECRET", "cn-north-4", false),
		api.WithHTTPClient(&http.Client{Transport: fn}),
		api.WithRetryPolicy(noopRetryPolicy{}),
		api.WithClock(func() time.Time { return time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC) }),
	)
}

func jsonResponse(r *http.Request, body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    r,
	}
}

const sampleZones = `{"zones":[
  {"id":"z-public-1","name":"ctk-demo.example.com.","status":"ACTIVE","zone_type":"public","record_num":4},
  {"id":"z-public-2","name":"another.example.","status":"ACTIVE","zone_type":"public","record_num":1}
],"metadata":{"total_count":2}}`

const sampleRecordSetsZ1 = `{"recordsets":[
  {"id":"r-a","name":"ctk-demo.example.com.","type":"A","ttl":300,"status":"ACTIVE","records":["198.51.100.10","198.51.100.11"]},
  {"id":"r-cname","name":"www.ctk-demo.example.com.","type":"CNAME","ttl":60,"status":"ACTIVE","records":["ctk-demo.example.com."]},
  {"id":"r-mx","name":"ctk-demo.example.com.","type":"MX","ttl":300,"status":"ACTIVE","records":["10 mx1.ctk-demo.example.com."]},
  {"id":"r-ns","name":"ctk-demo.example.com.","type":"NS","ttl":172800,"status":"ACTIVE","records":["ns1.example.com."]}
],"metadata":{"total_count":4}}`

const sampleRecordSetsZ2 = `{"recordsets":[
  {"id":"r-a-2","name":"db.another.example.","type":"A","ttl":60,"status":"ACTIVE","records":["10.0.0.10"]}
],"metadata":{"total_count":1}}`

func TestGetDomainsListsZonesAndRecordSets(t *testing.T) {
	driver := &Driver{
		Cred:    auth.New("AKID", "SECRET", "cn-north-4", false),
		Regions: []string{"cn-north-4"},
		Client: newTestClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.URL.Host == "dns.cn-north-4.myhuaweicloud.com" && r.URL.Path == "/v2/zones":
				return jsonResponse(r, sampleZones), nil
			case r.URL.Path == "/v2/zones/z-public-1/recordsets":
				return jsonResponse(r, sampleRecordSetsZ1), nil
			case r.URL.Path == "/v2/zones/z-public-2/recordsets":
				return jsonResponse(r, sampleRecordSetsZ2), nil
			default:
				t.Fatalf("unexpected request: %s %s%s", r.Method, r.URL.Host, r.URL.Path)
				return nil, nil
			}
		})),
	}
	domains, err := driver.GetDomains(context.Background())
	if err != nil {
		t.Fatalf("GetDomains: %v", err)
	}
	if len(domains) != 2 {
		t.Fatalf("expected 2 zones, got %d", len(domains))
	}
	z1 := domains[0]
	if z1.DomainName != "ctk-demo.example.com" {
		t.Errorf("expected trimmed FQDN, got %q", z1.DomainName)
	}
	// 2 A + 1 CNAME + 1 MX = 4 records (NS filtered)
	if len(z1.Records) != 4 {
		t.Fatalf("expected 4 records on zone 1, got %d (%+v)", len(z1.Records), z1.Records)
	}
	for _, rec := range z1.Records {
		if rec.Type == "NS" {
			t.Errorf("NS record should be filtered: %+v", rec)
		}
	}
	if z2 := domains[1]; z2.DomainName != "another.example" || len(z2.Records) != 1 || z2.Records[0].Value != "10.0.0.10" {
		t.Errorf("zone 2 mismatch: %+v", z2)
	}
}

func TestGetDomainsContinuesPastRecordSetFailure(t *testing.T) {
	driver := &Driver{
		Cred:    auth.New("AKID", "SECRET", "cn-north-4", false),
		Regions: []string{"cn-north-4"},
		Client: newTestClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.URL.Path == "/v2/zones":
				return jsonResponse(r, sampleZones), nil
			case r.URL.Path == "/v2/zones/z-public-1/recordsets":
				return &http.Response{
					StatusCode: http.StatusForbidden,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"error_code":"DNS.0403","error_msg":"forbidden"}`)),
					Request:    r,
				}, nil
			case r.URL.Path == "/v2/zones/z-public-2/recordsets":
				return jsonResponse(r, sampleRecordSetsZ2), nil
			default:
				t.Fatalf("unexpected: %s", r.URL.Path)
				return nil, nil
			}
		})),
	}
	domains, err := driver.GetDomains(context.Background())
	if err != nil {
		t.Fatalf("GetDomains returned fatal err on per-zone failure: %v", err)
	}
	if len(domains) != 2 {
		t.Fatalf("expected zones preserved, got %d", len(domains))
	}
	if len(domains[0].Records) != 0 {
		t.Errorf("expected denied zone to have empty records, got %+v", domains[0].Records)
	}
	if len(domains[1].Records) != 1 {
		t.Errorf("expected zone 2 records preserved, got %+v", domains[1].Records)
	}
}

func TestGetDomainsPropagatesZoneListError(t *testing.T) {
	driver := &Driver{
		Cred: auth.New("AKID", "SECRET", "cn-north-4", false),
		Client: newTestClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"error_code":"APIGW.0301","error_msg":"signature invalid"}`)),
				Request:    r,
			}, nil
		})),
	}
	_, err := driver.GetDomains(context.Background())
	if err == nil {
		t.Fatalf("expected error when ListZones fails")
	}
	if !strings.Contains(err.Error(), "APIGW.0301") {
		t.Errorf("expected APIGW.0301 in error, got %v", err)
	}
}
