package vm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

func TestDriverGetResourceRegionFallbacks(t *testing.T) {
	tests := []struct {
		name       string
		region     string
		wantPath   string
		wantRegion string
	}{
		{name: "default", region: "", wantPath: "/v1/regions/cn-north-1/instances", wantRegion: "cn-north-1"},
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

func TestDriverGetResourceEnumeratesAllKnownRegions(t *testing.T) {
	seen := make(map[string]int)
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		region := regionFromInstancesPath(t, r.URL.Path)
		if got := r.URL.Query().Get("pageNumber"); got != "1" {
			t.Fatalf("unexpected pageNumber: %s", got)
		}
		if got := r.URL.Query().Get("pageSize"); got != "100" {
			t.Fatalf("unexpected pageSize: %s", got)
		}
		mu.Lock()
		seen[region]++
		mu.Unlock()
		_, _ = w.Write([]byte(`{"requestId":"req-` + region + `","result":{"instances":[{"instanceId":"i-` + region + `","hostname":"` + region + `","status":"running","osType":"linux","privateIpAddress":"10.0.0.2","elasticIpAddress":"1.1.1.1"}]}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "all"}
	got, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}
	if len(got) != len(knownJDCloudVMRegions) {
		t.Fatalf("unexpected hosts: %+v", got)
	}
	if len(seen) != len(knownJDCloudVMRegions) {
		t.Fatalf("unexpected regions seen: %+v", seen)
	}
	for _, region := range knownJDCloudVMRegions {
		if seen[region] != 1 {
			t.Fatalf("expected one request for %s, got %d", region, seen[region])
		}
	}
}

func TestDriverGetResourceSeedsHostCache(t *testing.T) {
	SetCacheHostList(nil)
	defer SetCacheHostList(nil)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/regions/cn-north-1/instances" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"requestId":"req-cache","result":{"instances":[{"instanceId":"i-cache","hostname":"cache","status":"running","osType":"linux","privateIpAddress":"10.0.0.9","elasticIpAddress":"1.2.3.4"}]}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL)}
	if _, err := driver.GetResource(context.Background()); err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}

	cached := GetCacheHostList()
	if len(cached) != 1 || cached[0].ID != "i-cache" || cached[0].Region != "cn-north-1" {
		t.Fatalf("unexpected cached hosts: %+v", cached)
	}
}

func TestDriverGetCacheHostListReturnsSnapshot(t *testing.T) {
	hosts := []schema.Host{{ID: "i-1", Region: "cn-north-1"}}
	SetCacheHostList(hosts)
	defer SetCacheHostList(nil)

	hosts[0].ID = "mutated-before-read"
	first := GetCacheHostList()
	first[0].ID = "mutated-copy"

	second := GetCacheHostList()
	if len(second) != 1 || second[0].ID != "i-1" {
		t.Fatalf("expected cached snapshot isolation, got %+v", second)
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

func TestDriverGetResourceKeepsPartialItemsOnPaginationError(t *testing.T) {
	SetCacheHostList(nil)
	defer SetCacheHostList(nil)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/regions/cn-north-1/instances" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		pageNumber, err := strconv.Atoi(r.URL.Query().Get("pageNumber"))
		if err != nil {
			t.Fatalf("parse pageNumber: %v", err)
		}
		switch pageNumber {
		case 1:
			_, _ = w.Write([]byte(`{"requestId":"req-vm-1","result":{"instances":[{"instanceId":"i-1","hostname":"demo-1","status":"running","osType":"linux","privateIpAddress":"10.0.0.2","elasticIpAddress":"1.1.1.1"}],"totalCount":2}}`))
		case 2:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"requestId":"req-vm-2","error":{"status":"INTERNAL","code":500,"message":"temporary failure"}}`))
		default:
			t.Fatalf("unexpected pageNumber: %d", pageNumber)
		}
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL)}
	got, err := driver.GetResource(context.Background())
	if err == nil {
		t.Fatal("expected pagination error")
	}
	if len(got) != 1 || got[0].ID != "i-1" {
		t.Fatalf("expected partial hosts, got %+v", got)
	}

	cached := GetCacheHostList()
	if len(cached) != 1 || cached[0].ID != "i-1" {
		t.Fatalf("expected partial cache update, got %+v", cached)
	}
}

func TestDriverGetResourceDoesNotClearExistingCacheOnTotalFailure(t *testing.T) {
	SetCacheHostList([]schema.Host{{ID: "i-old", Region: "cn-north-1"}})
	defer SetCacheHostList(nil)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"requestId":"req-vm","error":{"status":"INTERNAL","code":500,"message":"temporary failure"}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL)}
	got, err := driver.GetResource(context.Background())
	if err == nil {
		t.Fatal("expected request error")
	}
	if len(got) != 0 {
		t.Fatalf("expected no new hosts, got %+v", got)
	}

	cached := GetCacheHostList()
	if len(cached) != 1 || cached[0].ID != "i-old" {
		t.Fatalf("expected previous cache preserved, got %+v", cached)
	}
}

func regionFromInstancesPath(t *testing.T, path string) string {
	t.Helper()
	const prefix = "/v1/regions/"
	const suffix = "/instances"
	if len(path) <= len(prefix)+len(suffix) || path[:len(prefix)] != prefix || path[len(path)-len(suffix):] != suffix {
		t.Fatalf("unexpected instances path: %s", path)
	}
	return path[len(prefix) : len(path)-len(suffix)]
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
