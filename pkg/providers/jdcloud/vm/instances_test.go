package vm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/auth"
)

func TestDriverGetResourceRegionFallbacks(t *testing.T) {
	tests := []struct {
		name       string
		region     string
		wantPath   string
		wantRegion string
	}{
		{name: "default", region: "", wantPath: "/v1/regions/cn-north-1/instances", wantRegion: "cn-north-1"},
		{name: "all", region: "all", wantPath: "/v1/regions/cn-north-1/instances", wantRegion: "cn-north-1"},
		{name: "explicit", region: "cn-east-2", wantPath: "/v1/regions/cn-east-2/instances", wantRegion: "cn-east-2"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.wantPath {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
				if got := r.URL.Query().Get("pageNumber"); got != "1" {
					t.Fatalf("unexpected pageNumber: %s", got)
				}
				if got := r.URL.Query().Get("pageSize"); got != "100" {
					t.Fatalf("unexpected pageSize: %s", got)
				}
				_, _ = w.Write([]byte(`{"requestId":"req-vm","result":{"instances":[{"instanceId":"i-1","hostname":"demo","status":"running","osType":"linux","privateIpAddress":"10.0.0.2","elasticIpAddress":"1.1.1.1"},{"instanceId":"i-2","hostname":"private","status":"stopped","osType":"linux","privateIpAddress":"10.0.0.3","elasticIpAddress":""}]}}`))
			}))
			defer server.Close()

			driver := &Driver{Client: newTestClient(server.URL), Region: tt.region}
			got, err := driver.GetResource(context.Background())
			if err != nil {
				t.Fatalf("GetResource() error = %v", err)
			}
			if len(got) != 2 {
				t.Fatalf("unexpected hosts: %+v", got)
			}
			if got[0].Region != tt.wantRegion || !got[0].Public || got[1].Public {
				t.Fatalf("unexpected mapped hosts: %+v", got)
			}
		})
	}
}

func TestDriverGetResourcePaginates(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/regions/cn-north-1/instances" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		requests++
		pageNumber, err := strconv.Atoi(r.URL.Query().Get("pageNumber"))
		if err != nil {
			t.Fatalf("parse pageNumber: %v", err)
		}
		if got := r.URL.Query().Get("pageSize"); got != "100" {
			t.Fatalf("unexpected pageSize: %s", got)
		}
		switch pageNumber {
		case 1:
			_, _ = w.Write([]byte(`{"requestId":"req-vm-1","result":{"instances":[{"instanceId":"i-1","hostname":"demo-1","status":"running","osType":"linux","privateIpAddress":"10.0.0.2","elasticIpAddress":"1.1.1.1"}],"totalCount":2}}`))
		case 2:
			_, _ = w.Write([]byte(`{"requestId":"req-vm-2","result":{"instances":[{"instanceId":"i-2","hostname":"demo-2","status":"stopped","osType":"linux","privateIpAddress":"10.0.0.3","elasticIpAddress":""}],"totalCount":2}}`))
		default:
			t.Fatalf("unexpected pageNumber: %d", pageNumber)
		}
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL)}
	got, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}
	if len(got) != 2 || got[0].ID != "i-1" || got[1].ID != "i-2" {
		t.Fatalf("unexpected hosts: %+v", got)
	}
	if requests != 2 {
		t.Fatalf("unexpected request count: %d", requests)
	}
}

func newTestClient(baseURL string) *api.Client {
	return api.NewClient(
		auth.New("AKID", "SECRET", "token64"),
		api.WithBaseURL(baseURL),
		api.WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
		api.WithNonceFunc(func() string { return "6691f002-da59-46eb-882c-edb66d46c917" }),
		api.WithRetryPolicy(api.RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)
}
