package lavm

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
				_, _ = w.Write([]byte(`{"requestId":"req-lavm","result":{"instances":[{"instanceId":"lavm-1","instanceName":"demo-lavm","status":"running","innerIpAddress":"10.0.0.2","publicIpAddress":"1.1.1.1","regionId":"` + tt.wantRegion + `","imageId":"img-linux","domains":[{"domainName":"demo.example.com"}]},{"instanceId":"lavm-2","instanceName":"private-lavm","status":"stopped","innerIpAddress":"10.0.0.3","publicIpAddress":"","regionId":"` + tt.wantRegion + `","imageId":"img-windows"}]}}`))
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
			if got[0].Region != tt.wantRegion || got[0].DNSName != "demo.example.com" || !got[0].Public || got[1].Public {
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
		_, _ = w.Write([]byte(`{"requestId":"req-` + region + `","result":{"instances":[{"instanceId":"lavm-` + region + `","instanceName":"` + region + `","status":"running","innerIpAddress":"10.0.0.2","publicIpAddress":"1.1.1.1","regionId":"` + region + `"}]}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "all"}
	got, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}
	if len(got) != len(knownJDCloudLAVMRegions) {
		t.Fatalf("unexpected hosts: %+v", got)
	}
	if len(seen) != len(knownJDCloudLAVMRegions) {
		t.Fatalf("unexpected regions seen: %+v", seen)
	}
	for _, region := range knownJDCloudLAVMRegions {
		if seen[region] != 1 {
			t.Fatalf("expected one request for %s, got %d", region, seen[region])
		}
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
			_, _ = w.Write([]byte(`{"requestId":"req-lavm-1","result":{"instances":[{"instanceId":"lavm-1","instanceName":"demo-1","status":"running","innerIpAddress":"10.0.0.2","publicIpAddress":"1.1.1.1","regionId":"cn-north-1"}],"totalCount":2}}`))
		case 2:
			_, _ = w.Write([]byte(`{"requestId":"req-lavm-2","result":{"instances":[{"instanceId":"lavm-2","instanceName":"demo-2","status":"stopped","innerIpAddress":"10.0.0.3","publicIpAddress":"","regionId":"cn-north-1"}],"totalCount":2}}`))
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
	if len(got) != 2 || got[0].ID != "lavm-1" || got[1].ID != "lavm-2" {
		t.Fatalf("unexpected hosts: %+v", got)
	}
	if requests != 2 {
		t.Fatalf("unexpected request count: %d", requests)
	}
}

func TestDriverGetResourceKeepsPartialItemsOnPaginationError(t *testing.T) {
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
			_, _ = w.Write([]byte(`{"requestId":"req-lavm-1","result":{"instances":[{"instanceId":"lavm-1","instanceName":"demo-1","status":"running","innerIpAddress":"10.0.0.2","publicIpAddress":"1.1.1.1","regionId":"cn-north-1"}],"totalCount":2}}`))
		case 2:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"requestId":"req-lavm-2","error":{"status":"INTERNAL","code":500,"message":"temporary failure"}}`))
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
	if len(got) != 1 || got[0].ID != "lavm-1" {
		t.Fatalf("expected partial hosts, got %+v", got)
	}
}

func TestDriverGetResourceSkipsInvalidRegionsInAllMode(t *testing.T) {
	originalRegions := knownJDCloudLAVMRegions
	knownJDCloudLAVMRegions = []string{"cn-east-2", "cn-north-1"}
	defer func() { knownJDCloudLAVMRegions = originalRegions }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		region := regionFromInstancesPath(t, r.URL.Path)
		switch region {
		case "cn-east-2":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"requestId":"req-invalid","error":{"status":"400","code":400,"message":"Invalid RegionId 'cn-east-2'"}}`))
		case "cn-north-1":
			_, _ = w.Write([]byte(`{"requestId":"req-ok","result":{"instances":[{"instanceId":"lavm-1","instanceName":"demo-1","status":"running","innerIpAddress":"10.0.0.2","publicIpAddress":"1.1.1.1","regionId":"cn-north-1"}]}}`))
		default:
			t.Fatalf("unexpected region: %s", region)
		}
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "all"}
	got, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("expected invalid region to be skipped, got %v", err)
	}
	if len(got) != 1 || got[0].ID != "lavm-1" {
		t.Fatalf("unexpected hosts: %+v", got)
	}
}

func TestNormalizeImageOSType(t *testing.T) {
	t.Skip("LAVM image-based osType enrichment is intentionally disabled for now")
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
