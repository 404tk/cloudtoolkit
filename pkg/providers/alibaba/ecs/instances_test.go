package ecs

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func TestGetResourceAllRegionsWithPaginationAndIPFallback(t *testing.T) {
	logger.SetOutput(io.Discard)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})
	SetCacheHostList(nil)
	t.Cleanup(func() {
		SetCacheHostList(nil)
	})

	var (
		mu    sync.Mutex
		calls []string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("Action")
		region := r.URL.Query().Get("RegionId")
		page := r.URL.Query().Get("PageNumber")

		mu.Lock()
		calls = append(calls, action+":"+region+":"+page)
		mu.Unlock()

		switch action {
		case "DescribeRegions":
			_, _ = io.WriteString(w, `{"RequestId":"req-regions","Regions":{"Region":[{"RegionId":"cn-hangzhou"},{"RegionId":"cn-shanghai"}]}}`)
		case "DescribeInstances":
			switch region + ":" + page {
			case "cn-hangzhou:1":
				_, _ = io.WriteString(w, `{"RequestId":"req-hz-1","TotalCount":2,"PageSize":1,"PageNumber":1,"Instances":{"Instance":[{"HostName":"web-1","InstanceId":"i-hz-1","OSType":"linux","PublicIpAddress":{"IpAddress":["1.1.1.1"]},"NetworkInterfaces":{"NetworkInterface":[{"PrimaryIpAddress":"10.0.0.9","PrivateIpSets":{"PrivateIpSet":[{"PrivateIpAddress":"10.0.0.1"}]}}]},"EipAddress":{"IpAddress":""}}]}}`)
			case "cn-hangzhou:2":
				_, _ = io.WriteString(w, `{"RequestId":"req-hz-2","TotalCount":2,"PageSize":1,"PageNumber":2,"Instances":{"Instance":[{"HostName":"web-2","InstanceId":"i-hz-2","OSType":"linux","PublicIpAddress":{"IpAddress":[]},"NetworkInterfaces":{"NetworkInterface":[{"PrimaryIpAddress":"10.0.0.2","PrivateIpSets":{"PrivateIpSet":[]}}]},"EipAddress":{"IpAddress":"2.2.2.2"}}]}}`)
			case "cn-shanghai:1":
				_, _ = io.WriteString(w, `{"RequestId":"req-sh-1","TotalCount":1,"PageSize":100,"PageNumber":1,"Instances":{"Instance":[{"HostName":"db-1","InstanceId":"i-sh-1","OSType":"windows","PublicIpAddress":{"IpAddress":[]},"NetworkInterfaces":{"NetworkInterface":[{"PrimaryIpAddress":"","PrivateIpSets":{"PrivateIpSet":[]}},{"PrimaryIpAddress":"10.1.0.3","PrivateIpSets":{"PrivateIpSet":[]}}]},"EipAddress":{"IpAddress":""}}]}}`)
			default:
				t.Fatalf("unexpected describe instances request: region=%s page=%s", region, page)
			}
		default:
			t.Fatalf("unexpected action: %s", action)
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	hosts, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}
	if len(hosts) != 3 {
		t.Fatalf("unexpected host count: %d", len(hosts))
	}

	if len(calls) != 4 {
		t.Fatalf("unexpected call count: %v", calls)
	}
	if calls[0] != "DescribeRegions:cn-hangzhou:" {
		t.Fatalf("expected DescribeRegions first, got %v", calls)
	}

	assertHost(t, hosts, schema.Host{
		HostName:    "web-1",
		ID:          "i-hz-1",
		PublicIPv4:  "1.1.1.1",
		PrivateIpv4: "10.0.0.1",
		OSType:      "linux",
		Public:      true,
		Region:      "cn-hangzhou",
	})
	assertHost(t, hosts, schema.Host{
		HostName:    "web-2",
		ID:          "i-hz-2",
		PublicIPv4:  "2.2.2.2",
		PrivateIpv4: "10.0.0.2",
		OSType:      "linux",
		Public:      true,
		Region:      "cn-hangzhou",
	})
	assertHost(t, hosts, schema.Host{
		HostName:    "db-1",
		ID:          "i-sh-1",
		PublicIPv4:  "",
		PrivateIpv4: "10.1.0.3",
		OSType:      "windows",
		Public:      false,
		Region:      "cn-shanghai",
	})

	cached := GetCacheHostList()
	if len(cached) != len(hosts) {
		t.Fatalf("unexpected cache host count: %d", len(cached))
	}
	assertHost(t, cached, schema.Host{
		HostName:    "web-1",
		ID:          "i-hz-1",
		PublicIPv4:  "1.1.1.1",
		PrivateIpv4: "10.0.0.1",
		OSType:      "linux",
		Public:      true,
		Region:      "cn-hangzhou",
	})
}

func TestGetResourceSingleRegionSkipsDescribeRegions(t *testing.T) {
	logger.SetOutput(io.Discard)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})
	SetCacheHostList(nil)
	t.Cleanup(func() {
		SetCacheHostList(nil)
	})

	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("Action")
		calls = append(calls, action)
		if action != "DescribeInstances" {
			t.Fatalf("unexpected action: %s", action)
		}
		if got := r.URL.Query().Get("RegionId"); got != "cn-beijing" {
			t.Fatalf("unexpected region: %s", got)
		}
		if got := r.URL.Query().Get("PageSize"); got != "100" {
			t.Fatalf("unexpected page size: %s", got)
		}
		_, _ = io.WriteString(w, `{"RequestId":"req-bj-1","TotalCount":1,"PageSize":100,"PageNumber":1,"Instances":{"Instance":[{"HostName":"api-1","InstanceId":"i-bj-1","OSType":"linux","PublicIpAddress":{"IpAddress":["8.8.8.8"]},"NetworkInterfaces":{"NetworkInterface":[{"PrimaryIpAddress":"172.16.0.8","PrivateIpSets":{"PrivateIpSet":[{"PrivateIpAddress":"172.16.0.7"}]}}]},"EipAddress":{"IpAddress":""}}]}}`)
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	driver.Region = "cn-beijing"

	hosts, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}
	if len(calls) != 1 || calls[0] != "DescribeInstances" {
		t.Fatalf("unexpected calls: %v", calls)
	}
	assertHost(t, hosts, schema.Host{
		HostName:    "api-1",
		ID:          "i-bj-1",
		PublicIPv4:  "8.8.8.8",
		PrivateIpv4: "172.16.0.7",
		OSType:      "linux",
		Public:      true,
		Region:      "cn-beijing",
	})
}

func newTestDriver(baseURL string) Driver {
	return Driver{
		Cred:   aliauth.New("ak", "sk", ""),
		Region: "all",
		clientOptions: []api.Option{
			api.WithBaseURL(baseURL),
			api.WithClock(func() time.Time { return time.Unix(1713376800, 0).UTC() }),
			api.WithNonce(func() string { return "nonce" }),
		},
	}
}

func assertHost(t *testing.T, hosts []schema.Host, want schema.Host) {
	t.Helper()

	for _, host := range hosts {
		if host.ID != want.ID {
			continue
		}
		if host != want {
			t.Fatalf("unexpected host for %s: got %+v want %+v", want.ID, host, want)
		}
		return
	}
	t.Fatalf("host %s not found in %+v", want.ID, hosts)
}
