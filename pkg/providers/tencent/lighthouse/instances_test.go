package lighthouse

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func TestGetResourceAllRegionsPaginatesAndMapsHosts(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger.SetOutput(buffer)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-TC-Action") {
		case "DescribeRegions":
			if got := r.Header.Get("X-TC-Region"); got != api.DefaultRegion {
				t.Fatalf("unexpected DescribeRegions region: %s", got)
			}
			if body := readBody(t, r); body != "{}" {
				t.Fatalf("unexpected DescribeRegions body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"RegionSet":[{"Region":"ap-guangzhou"},{"Region":"ap-singapore"}],"RequestId":"req-regions"}}`))
		case "DescribeInstances":
			switch region := r.Header.Get("X-TC-Region"); region {
			case "ap-guangzhou":
				switch body := readBody(t, r); body {
				case `{"Offset":0,"Limit":100}`:
					_, _ = w.Write([]byte(`{"Response":{"TotalCount":1,"InstanceSet":[{"InstanceId":"lh-gz-1","InstanceName":"lh-linux","InstanceState":"RUNNING","PublicAddresses":["3.3.3.3"],"PrivateAddresses":["172.16.0.1"],"PlatformType":"LINUX_UNIX"}],"RequestId":"req-gz-1"}}`))
				default:
					t.Fatalf("unexpected guangzhou body: %s", body)
				}
			case "ap-singapore":
				switch body := readBody(t, r); body {
				case `{"Offset":0,"Limit":100}`:
					_, _ = w.Write([]byte(`{"Response":{"TotalCount":2,"InstanceSet":[{"InstanceId":"lh-sg-1","InstanceName":"lh-win","InstanceState":"STOPPED","PrivateAddresses":["172.16.1.1"],"PlatformType":"WINDOWS"}],"RequestId":"req-sg-1"}}`))
				case `{"Offset":1,"Limit":100}`:
					_, _ = w.Write([]byte(`{"Response":{"TotalCount":2,"InstanceSet":[{"InstanceId":"lh-sg-2","InstanceName":"lh-no-ip","InstanceState":"RUNNING","PlatformType":"LINUX_UNIX"}],"RequestId":"req-sg-2"}}`))
				default:
					t.Fatalf("unexpected singapore body: %s", body)
				}
			default:
				t.Fatalf("unexpected DescribeInstances region: %s", region)
			}
		default:
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "all")
	hosts, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}
	if len(hosts) != 3 {
		t.Fatalf("unexpected host count: %d", len(hosts))
	}

	byID := map[string]struct {
		OSType     string
		PublicIPv4 string
		Region     string
	}{
		"lh-gz-1": {OSType: "LINUX_UNIX", PublicIPv4: "3.3.3.3", Region: "ap-guangzhou"},
		"lh-sg-1": {OSType: "WINDOWS", PublicIPv4: "", Region: "ap-singapore"},
		"lh-sg-2": {OSType: "LINUX_UNIX", PublicIPv4: "", Region: "ap-singapore"},
	}
	for _, host := range hosts {
		expect, ok := byID[host.ID]
		if !ok {
			t.Fatalf("unexpected host: %+v", host)
		}
		if host.OSType != expect.OSType || host.PublicIPv4 != expect.PublicIPv4 || host.Region != expect.Region {
			t.Fatalf("unexpected mapped host: %+v", host)
		}
	}
}

func TestGetResourceUsesDefaultRegionWhenUnset(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger.SetOutput(buffer)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})

	sawDescribeRegions := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-TC-Action") {
		case "DescribeRegions":
			sawDescribeRegions = true
			t.Fatalf("DescribeRegions should not be called when region is unset")
		case "DescribeInstances":
			if got := r.Header.Get("X-TC-Region"); got != api.DefaultRegion {
				t.Fatalf("unexpected default region: %s", got)
			}
			if body := readBody(t, r); body != `{"Offset":0,"Limit":100}` {
				t.Fatalf("unexpected DescribeInstances body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"TotalCount":1,"InstanceSet":[{"InstanceId":"lh-default","InstanceName":"default-lh","InstanceState":"RUNNING","PrivateAddresses":["172.16.2.1"],"PlatformType":"LINUX_UNIX"}],"RequestId":"req-default"}}`))
		default:
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "")
	hosts, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}
	if sawDescribeRegions {
		t.Fatal("unexpected DescribeRegions call")
	}
	if len(hosts) != 1 {
		t.Fatalf("unexpected host count: %d", len(hosts))
	}
	if hosts[0].Region != api.DefaultRegion {
		t.Fatalf("unexpected host region: %+v", hosts[0])
	}
	if hosts[0].Public {
		t.Fatalf("expected host without public address: %+v", hosts[0])
	}
}

func newTestDriver(baseURL, region string) Driver {
	return Driver{
		Credential: auth.New("ak", "sk", ""),
		Region:     region,
		clientOptions: []api.Option{
			api.WithBaseURL(baseURL),
			api.WithClock(func() time.Time { return time.Unix(1776458501, 0).UTC() }),
			api.WithRetryPolicy(api.RetryPolicy{
				MaxAttempts: 1,
				Sleep:       func(context.Context, time.Duration) error { return nil },
			}),
		},
	}
}

func readBody(t *testing.T, r *http.Request) string {
	t.Helper()
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return strings.TrimSpace(string(body))
}
