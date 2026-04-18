package cvm

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
			_, _ = w.Write([]byte(`{"Response":{"RegionSet":[{"Region":"ap-guangzhou"},{"Region":"ap-shanghai"}],"RequestId":"req-regions"}}`))
		case "DescribeInstances":
			switch region := r.Header.Get("X-TC-Region"); region {
			case "ap-guangzhou":
				switch body := readBody(t, r); body {
				case `{"Offset":0,"Limit":100}`:
					_, _ = w.Write([]byte(`{"Response":{"TotalCount":1,"InstanceSet":[{"InstanceId":"ins-gz-1","InstanceName":"gz-linux","InstanceState":"RUNNING","PublicIpAddresses":["1.1.1.1"],"PrivateIpAddresses":["10.0.0.1"],"OsName":"Ubuntu 22.04"}],"RequestId":"req-gz-1"}}`))
				default:
					t.Fatalf("unexpected guangzhou body: %s", body)
				}
			case "ap-shanghai":
				switch body := readBody(t, r); body {
				case `{"Offset":0,"Limit":100}`:
					_, _ = w.Write([]byte(`{"Response":{"TotalCount":2,"InstanceSet":[{"InstanceId":"ins-sh-1","InstanceName":"sh-win","InstanceState":"STOPPED","PublicIpAddresses":["2.2.2.2"],"PrivateIpAddresses":["10.0.1.1"],"OsName":"Windows Server 2022"}],"RequestId":"req-sh-1"}}`))
				case `{"Offset":1,"Limit":100}`:
					_, _ = w.Write([]byte(`{"Response":{"TotalCount":2,"InstanceSet":[{"InstanceId":"ins-sh-2","InstanceName":"sh-linux","InstanceState":"RUNNING","PrivateIpAddresses":["10.0.1.2"],"OsName":"CentOS 7.9"}],"RequestId":"req-sh-2"}}`))
				default:
					t.Fatalf("unexpected shanghai body: %s", body)
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
		OSType string
		Public bool
		Region string
	}{
		"ins-gz-1": {OSType: "LINUX_UNIX", Public: true, Region: "ap-guangzhou"},
		"ins-sh-1": {OSType: "WINDOWS", Public: true, Region: "ap-shanghai"},
		"ins-sh-2": {OSType: "LINUX_UNIX", Public: false, Region: "ap-shanghai"},
	}
	for _, host := range hosts {
		expect, ok := byID[host.ID]
		if !ok {
			t.Fatalf("unexpected host: %+v", host)
		}
		if host.OSType != expect.OSType || host.Public != expect.Public || host.Region != expect.Region {
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
			_, _ = w.Write([]byte(`{"Response":{"TotalCount":1,"InstanceSet":[{"InstanceId":"ins-default","InstanceName":"default-host","InstanceState":"RUNNING","PrivateIpAddresses":["10.0.2.1"],"OsName":"Debian 12"}],"RequestId":"req-default"}}`))
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
	if hosts[0].PublicIPv4 != "" || hosts[0].Public {
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
