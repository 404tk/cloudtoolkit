package rds

import (
	"bytes"
	"context"
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

func TestDriverGetDatabasesListsAcrossPagesAndPrintsProgress(t *testing.T) {
	transport := &routingTransport{
		t: t,
		routes: map[string]routeResponse{
			"GET iam.cn-north-4.myhuaweicloud.com /v3/projects?name=cn-north-4": {
				body: `{"projects":[{"id":"project-n4","name":"cn-north-4","domain_id":"d-1","enabled":true}]}`,
			},
			"GET rds.cn-north-4.myhuaweicloud.com /v3/project-n4/instances?limit=100&offset=0": {
				body: `{"instances":[{"id":"db-1","region":"cn-north-4","port":3306,"public_ips":["1.1.1.1"],"private_ips":["10.0.0.1"],"datastore":{"type":"MySQL","version":"8.0"}}],"total_count":101}`,
			},
			"GET rds.cn-north-4.myhuaweicloud.com /v3/project-n4/instances?limit=100&offset=100": {
				body: `{"instances":[{"id":"db-2","region":"","port":5432,"private_ips":["10.0.0.2"],"datastore":{"type":"PostgreSQL","version":"14"}}]}`,
			},
		},
	}

	driver := newTestDriver([]string{"cn-north-4"}, "d-1", transport)
	var (
		got    []schema.Database
		err    error
		output string
	)
	output = captureStdout(t, func() {
		got, err = driver.GetDatabases(context.Background())
	})
	if err != nil {
		t.Fatalf("GetDatabases() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("unexpected database count: %d", len(got))
	}
	if !strings.Contains(output, "[cn-north-4] 2 found.") {
		t.Fatalf("unexpected progress output: %q", output)
	}
	if got[0].InstanceId != "db-1" || got[0].Engine != "MySQL" || got[0].EngineVersion != "8.0" || got[0].Region != "cn-north-4" || got[0].Address != "1.1.1.1:3306" {
		t.Fatalf("unexpected first database: %+v", got[0])
	}
	if got[1].InstanceId != "db-2" || got[1].Engine != "PostgreSQL" || got[1].EngineVersion != "14" || got[1].Region != "cn-north-4" || got[1].Address != "10.0.0.2:5432" {
		t.Fatalf("unexpected second database: %+v", got[1])
	}
}

func TestDriverGetDatabasesAggregatesRegionErrorsAndContinues(t *testing.T) {
	transport := &routingTransport{
		t: t,
		routes: map[string]routeResponse{
			"GET iam.cn-north-4.myhuaweicloud.com /v3/projects?name=cn-north-4": {
				body: `{"projects":[{"id":"project-n4","name":"cn-north-4","domain_id":"d-1","enabled":true}]}`,
			},
			"GET rds.cn-north-4.myhuaweicloud.com /v3/project-n4/instances?limit=100&offset=0": {
				statusCode: http.StatusBadRequest,
				body:       `{"error_code":"RDS.0001","error_msg":"broken"}`,
			},
			"GET iam.cn-north-4.myhuaweicloud.com /v3/projects?name=cn-east-3": {
				body: `{"projects":[{"id":"project-e3","name":"cn-east-3","domain_id":"d-1","enabled":true}]}`,
			},
			"GET rds.cn-east-3.myhuaweicloud.com /v3/project-e3/instances?limit=100&offset=0": {
				body: `{"instances":[{"id":"db-2","region":"cn-east-3","port":5432,"private_ips":["10.0.0.2"],"datastore":{"type":"PostgreSQL","version":"14"}}],"total_count":1}`,
			},
		},
	}

	driver := newTestDriver([]string{"cn-north-4", "cn-east-3"}, "d-1", transport)
	got, err := driver.GetDatabases(context.Background())
	if err == nil {
		t.Fatal("expected aggregated error")
	}
	if len(got) != 1 || got[0].InstanceId != "db-2" {
		t.Fatalf("unexpected partial databases: %+v", got)
	}
	if !strings.Contains(err.Error(), "cn-north-4") || !strings.Contains(err.Error(), "broken") {
		t.Fatalf("unexpected aggregated error: %v", err)
	}
}

func TestDriverGetDatabasesSkipsRegionsWithoutProject(t *testing.T) {
	transport := &routingTransport{
		t: t,
		routes: map[string]routeResponse{
			"GET iam.cn-north-4.myhuaweicloud.com /v3/projects?name=cn-east-201": {
				body: `{"projects":[]}`,
			},
			"GET iam.cn-north-4.myhuaweicloud.com /v3/projects?name=cn-east-3": {
				body: `{"projects":[{"id":"project-e3","name":"cn-east-3","domain_id":"d-1","enabled":true}]}`,
			},
			"GET rds.cn-east-3.myhuaweicloud.com /v3/project-e3/instances?limit=100&offset=0": {
				body: `{"instances":[{"id":"db-2","region":"cn-east-3","port":5432,"private_ips":["10.0.0.2"],"datastore":{"type":"PostgreSQL","version":"14"}}],"total_count":1}`,
			},
		},
	}

	driver := newTestDriver([]string{"cn-east-201", "cn-east-3"}, "d-1", transport)
	got, err := driver.GetDatabases(context.Background())
	if err != nil {
		t.Fatalf("GetDatabases() error = %v", err)
	}
	if len(got) != 1 || got[0].InstanceId != "db-2" {
		t.Fatalf("unexpected databases: %+v", got)
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
	t      *testing.T
	routes map[string]routeResponse
}

func (rt *routingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	rt.t.Helper()
	if got := r.Header.Get(api.HeaderAuthorization); got == "" {
		rt.t.Fatal("missing authorization header")
	}
	key := r.Method + " " + r.URL.Host + " " + r.URL.Path + "?" + r.URL.RawQuery
	resp, ok := rt.routes[key]
	if !ok {
		rt.t.Fatalf("unexpected request: %s", key)
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
