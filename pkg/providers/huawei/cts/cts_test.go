package cts

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

func TestDumpEventsFiltersBySourceIPAndFollowsMarker(t *testing.T) {
	t.Parallel()

	requests := 0
	driver := &Driver{
		Cred:     auth.New("AKID", "SECRET", "cn-north-4", false),
		Regions:  []string{"cn-north-4"},
		DomainID: "d-1",
		Client: newTestClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.URL.Host == "iam.cn-north-4.myhuaweicloud.com" && r.URL.Path == "/v3/projects":
				if got := r.URL.Query().Get("name"); got != "cn-north-4" {
					t.Fatalf("unexpected region lookup name: %s", got)
				}
				return jsonResponse(r, `{"projects":[{"id":"project-n4","name":"cn-north-4","domain_id":"d-1","enabled":true}]}`), nil
			case r.URL.Host == "cts.cn-north-4.myhuaweicloud.com" && r.URL.Path == "/v3/project-n4/traces":
				requests++
				if got := r.URL.Query().Get("trace_type"); got != "system" {
					t.Fatalf("unexpected trace_type: %s", got)
				}
				if got := r.URL.Query().Get("tracker_name"); got != "system" {
					t.Fatalf("unexpected tracker_name: %s", got)
				}
				if got := r.URL.Query().Get("limit"); got != "200" {
					t.Fatalf("unexpected limit: %s", got)
				}
				if requests == 1 {
					if got := r.URL.Query().Get("next"); got != "" {
						t.Fatalf("unexpected next on first page: %s", got)
					}
					return jsonResponse(r, `{"traces":[
						{"trace_id":"trace-1","trace_name":"deleteEip","operation_id":"DeletePublicip","resource_name":"eip-prod","source_ip":"203.0.113.10","code":"204","time":1740710091805,"user":{"access_key_id":"AK-1"}},
						{"trace_id":"trace-2","trace_name":"getResourceTags","operation_id":"GetResourceTags","resource_name":"-","resource_id":"res-2","source_ip":"198.51.100.15","code":"200","time":1740710053352,"user":{"access_key_id":"AK-2"}}
					],"meta_data":{"count":2,"marker":"page-2"}}`), nil
				}
				if got := r.URL.Query().Get("next"); got != "page-2" {
					t.Fatalf("unexpected next on second page: %s", got)
				}
				return jsonResponse(r, `{"traces":[
					{"trace_id":"trace-3","trace_name":"createUser","operation_id":"CreateUser","resource_name":"ctk-demo-bot","source_ip":"203.0.113.10","code":"201","time":1740710191805,"user":{"access_key_id":"AK-3"}}
				],"meta_data":{"count":1}}`), nil
			default:
				t.Fatalf("unexpected request: %s %s%s", r.Method, r.URL.Host, r.URL.Path)
				return nil, nil
			}
		})),
	}

	got, err := driver.DumpEvents(context.Background(), "203.0.113.10")
	if err != nil {
		t.Fatalf("DumpEvents() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 filtered events, got %d", len(got))
	}
	if got[0].Id != "trace-1" || got[0].Affected != "eip-prod" || got[0].API != "DeletePublicip" || got[0].Status != "成功" {
		t.Fatalf("unexpected first event: %#v", got[0])
	}
	if got[1].Id != "trace-3" || got[1].AccessKey != "AK-3" {
		t.Fatalf("unexpected second event: %#v", got[1])
	}
	if got[0].Time != "2025-02-28T02:34:51Z" {
		t.Fatalf("unexpected event time: %s", got[0].Time)
	}
}

func TestDumpEventsSkipsProjectNotFoundRegions(t *testing.T) {
	t.Parallel()

	driver := &Driver{
		Cred:     auth.New("AKID", "SECRET", "cn-north-4", false),
		Regions:  []string{"cn-east-201", "cn-north-4"},
		DomainID: "d-1",
		Client: newTestClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.URL.Host == "iam.cn-north-4.myhuaweicloud.com" && r.URL.Path == "/v3/projects" && r.URL.Query().Get("name") == "cn-east-201":
				return jsonResponse(r, `{"projects":[]}`), nil
			case r.URL.Host == "iam.cn-north-4.myhuaweicloud.com" && r.URL.Path == "/v3/projects" && r.URL.Query().Get("name") == "cn-north-4":
				return jsonResponse(r, `{"projects":[{"id":"project-n4","name":"cn-north-4","domain_id":"d-1","enabled":true}]}`), nil
			case r.URL.Host == "cts.cn-north-4.myhuaweicloud.com" && r.URL.Path == "/v3/project-n4/traces":
				return jsonResponse(r, `{"traces":[
					{"trace_id":"trace-1","trace_name":"deleteEip","operation_id":"DeletePublicip","resource_name":"eip-prod","source_ip":"203.0.113.10","code":"204","time":1740710091805,"user":{"access_key_id":"AK-1"}}
				],"meta_data":{"count":1}}`), nil
			default:
				t.Fatalf("unexpected request: %s %s%s", r.Method, r.URL.Host, r.URL.Path)
				return nil, nil
			}
		})),
	}

	got, err := driver.DumpEvents(context.Background(), "all")
	if err != nil {
		t.Fatalf("DumpEvents() error = %v", err)
	}
	if len(got) != 1 || got[0].Id != "trace-1" {
		t.Fatalf("unexpected events: %#v", got)
	}
}

func TestHandleEventsReturnsUnsupported(t *testing.T) {
	t.Parallel()

	driver := &Driver{}
	got, err := driver.HandleEvents(context.Background(), "evt-1")
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if got.Action != "" || got.Scope != "" || len(got.Events) != 0 || got.TaskID != 0 || got.Message != "" {
		t.Fatalf("expected empty result, got %#v", got)
	}
	if !strings.Contains(err.Error(), "read-only") {
		t.Fatalf("unexpected error: %v", err)
	}
}
