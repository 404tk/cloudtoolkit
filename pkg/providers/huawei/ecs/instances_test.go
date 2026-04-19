package ecs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

func TestDriverGetResourceMapsHostsAcrossPages(t *testing.T) {
	transport := &routingTransport{
		routes: map[string]routeResponse{
			"GET iam.cn-north-4.myhuaweicloud.com /v3/projects?name=cn-north-4": {
				body: `{"projects":[{"id":"project-n4","name":"cn-north-4","domain_id":"d-1","enabled":true}]}`,
			},
			"GET ecs.cn-north-4.myhuaweicloud.com /v1/project-n4/cloudservers/detail?limit=100&offset=1": {
				body: `{"count":101,"servers":[{"status":"ACTIVE","name":"ecs-1","addresses":{"net-a":[{"addr":"10.0.0.1","OS-EXT-IPS:type":"fixed"},{"addr":"1.1.1.1","OS-EXT-IPS:type":"floating"}]}}]}`,
			},
			"GET ecs.cn-north-4.myhuaweicloud.com /v1/project-n4/cloudservers/detail?limit=100&offset=2": {
				body: `{"count":101,"servers":[{"status":"SHUTOFF","name":"ecs-2","addresses":{"net-a":[{"addr":"10.0.0.2","OS-EXT-IPS:type":"fixed"}]}}]}`,
			},
		},
	}

	driver := newTestDriver([]string{"cn-north-4"}, "d-1", transport)
	var (
		got []schema.Host
		err error
	)
	_ = captureStdout(t, func() {
		got, err = driver.GetResource(context.Background())
	})
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("unexpected host count: %d", len(got))
	}
	if got[0].HostName != "ecs-1" || got[0].State != "ACTIVE" || got[0].PublicIPv4 != "1.1.1.1" || got[0].PrivateIpv4 != "10.0.0.1" || !got[0].Public || got[0].Region != "cn-north-4" {
		t.Fatalf("unexpected first host: %+v", got[0])
	}
	if got[1].HostName != "ecs-2" || got[1].State != "SHUTOFF" || got[1].PublicIPv4 != "" || got[1].PrivateIpv4 != "10.0.0.2" || got[1].Public || got[1].Region != "cn-north-4" {
		t.Fatalf("unexpected second host: %+v", got[1])
	}
}

func TestDriverGetResourceAggregatesRegionErrorsAndContinues(t *testing.T) {
	transport := &routingTransport{
		routes: map[string]routeResponse{
			"GET iam.cn-north-4.myhuaweicloud.com /v3/projects?name=cn-north-4": {
				body: `{"projects":[{"id":"project-n4","name":"cn-north-4","domain_id":"d-1","enabled":true}]}`,
			},
			"GET ecs.cn-north-4.myhuaweicloud.com /v1/project-n4/cloudservers/detail?limit=100&offset=1": {
				statusCode: http.StatusBadRequest,
				body:       `{"error_code":"ECS.0001","error_msg":"broken"}`,
			},
			"GET iam.cn-north-4.myhuaweicloud.com /v3/projects?name=cn-east-3": {
				body: `{"projects":[{"id":"project-e3","name":"cn-east-3","domain_id":"d-1","enabled":true}]}`,
			},
			"GET ecs.cn-east-3.myhuaweicloud.com /v1/project-e3/cloudservers/detail?limit=100&offset=1": {
				body: `{"count":1,"servers":[{"status":"ACTIVE","name":"ecs-ok","addresses":{"net-a":[{"addr":"10.0.0.3","OS-EXT-IPS:type":"fixed"}]}}]}`,
			},
		},
	}

	driver := newTestDriver([]string{"cn-north-4", "cn-east-3"}, "d-1", transport)
	var (
		got []schema.Host
		err error
	)
	_ = captureStdout(t, func() {
		got, err = driver.GetResource(context.Background())
	})
	if err == nil {
		t.Fatal("expected aggregated error")
	}
	if len(got) != 1 || got[0].HostName != "ecs-ok" {
		t.Fatalf("unexpected partial hosts: %+v", got)
	}
	if !strings.Contains(err.Error(), "cn-north-4") || !strings.Contains(err.Error(), "broken") {
		t.Fatalf("unexpected aggregated error: %v", err)
	}
}

func newTestDriver(regions []string, domainID string, transport http.RoundTripper) *Driver {
	cred := auth.New("AKID", "SECRET", "cn-north-4", false)
	return &Driver{
		Cred:     cred,
		Regions:  regions,
		DomainID: domainID,
		Client: api.NewClient(
			cred,
			api.WithHTTPClient(&http.Client{Transport: transport}),
			api.WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
			api.WithRetryPolicy(noopRetryPolicy{}),
		),
	}
}

type routeResponse struct {
	statusCode int
	body       string
}

type routingTransport struct {
	routes map[string]routeResponse
}

func (rt *routingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if got := r.Header.Get(api.HeaderAuthorization); got == "" {
		return nil, fmt.Errorf("missing authorization header")
	}
	key := r.Method + " " + r.URL.Host + " " + r.URL.Path + "?" + r.URL.RawQuery
	resp, ok := rt.routes[key]
	if !ok {
		return nil, fmt.Errorf("unexpected request: %s", key)
	}
	statusCode := resp.statusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	return &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(resp.body)),
		Request:    r,
	}, nil
}

type noopRetryPolicy struct{}

func (noopRetryPolicy) Do(ctx context.Context, _ bool, fn func() (*http.Response, error)) (*http.Response, error) {
	return fn()
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()

	_ = w.Close()
	os.Stdout = oldStdout
	output := <-done
	_ = r.Close()
	return output
}
