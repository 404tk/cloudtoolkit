package ec2

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/aws/auth"
)

func TestDriverGetEC2RegionsUsesBootstrapRegionForAll(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values := mustParseEC2BodyValues(t, r)
		if got := values.Get("Action"); got != "DescribeRegions" {
			t.Fatalf("unexpected action: %s", got)
		}
		if got := signingRegionFromAuthorization(t, r.Header.Get("Authorization")); got != "cn-northwest-1" {
			t.Fatalf("unexpected signing region: %s", got)
		}
		_, _ = w.Write([]byte(`
<DescribeRegionsResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
  <regionInfo>
    <item><regionName>cn-northwest-1</regionName></item>
    <item><regionName>cn-north-1</regionName></item>
  </regionInfo>
</DescribeRegionsResponse>`))
	}))
	defer server.Close()

	driver := &Driver{
		Client:        newEC2DriverTestClient(server.URL),
		Region:        "all",
		DefaultRegion: "cn-northwest-1",
	}
	got, err := driver.GetEC2Regions(context.Background())
	if err != nil {
		t.Fatalf("GetEC2Regions() error = %v", err)
	}
	if len(got) != 2 || got[0] != "cn-northwest-1" || got[1] != "cn-north-1" {
		t.Fatalf("unexpected regions: %+v", got)
	}
}

func TestDriverGetResourceMapsInstancesAcrossPagesAndRegions(t *testing.T) {
	var (
		mu        sync.Mutex
		pageCount = map[string]int{}
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values := mustParseEC2BodyValues(t, r)
		switch values.Get("Action") {
		case "DescribeRegions":
			_, _ = w.Write([]byte(`
<DescribeRegionsResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
  <regionInfo>
    <item><regionName>ap-southeast-1</regionName></item>
    <item><regionName>ap-east-1</regionName></item>
  </regionInfo>
</DescribeRegionsResponse>`))
		case "DescribeInstances":
			region := signingRegionFromAuthorization(t, r.Header.Get("Authorization"))
			mu.Lock()
			pageCount[region]++
			page := pageCount[region]
			mu.Unlock()
			switch region {
			case "ap-southeast-1":
				if page == 1 {
					if got := values.Get("NextToken"); got != "" {
						t.Fatalf("unexpected first page token: %s", got)
					}
					_, _ = w.Write([]byte(`
<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
  <reservationSet>
    <item>
      <instancesSet>
        <item>
          <instanceId>i-sg-1</instanceId>
          <ipAddress>1.1.1.1</ipAddress>
          <privateIpAddress>10.0.0.1</privateIpAddress>
          <dnsName>ec2-1-1-1-1.compute.amazonaws.com</dnsName>
          <instanceState><name>running</name></instanceState>
          <tagSet>
            <item><key>Name</key><value>name-fallback</value></item>
            <item><key>aws:cloudformation:stack-name</key><value>stack-preferred</value></item>
          </tagSet>
        </item>
      </instancesSet>
    </item>
  </reservationSet>
  <nextToken>page-2</nextToken>
</DescribeInstancesResponse>`))
					return
				}
				if got := values.Get("NextToken"); got != "page-2" {
					t.Fatalf("unexpected second page token: %s", got)
				}
				_, _ = w.Write([]byte(`
<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
  <reservationSet>
    <item>
      <instancesSet>
        <item>
          <instanceId>i-sg-2</instanceId>
          <privateIpAddress>10.0.0.2</privateIpAddress>
          <instanceState><name>stopped</name></instanceState>
        </item>
      </instancesSet>
    </item>
  </reservationSet>
</DescribeInstancesResponse>`))
			case "ap-east-1":
				_, _ = w.Write([]byte(`
<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
  <reservationSet>
    <item>
      <instancesSet>
        <item>
          <instanceId>i-east-1</instanceId>
          <ipAddress>2.2.2.2</ipAddress>
          <privateIpAddress>10.1.0.1</privateIpAddress>
          <dnsName>ec2-2-2-2-2.compute.amazonaws.com</dnsName>
          <instanceState><name>running</name></instanceState>
          <tagSet>
            <item><key>Name</key><value>east-host</value></item>
          </tagSet>
        </item>
      </instancesSet>
    </item>
  </reservationSet>
</DescribeInstancesResponse>`))
			default:
				t.Fatalf("unexpected region: %s", region)
			}
		default:
			t.Fatalf("unexpected action: %s", values.Get("Action"))
		}
	}))
	defer server.Close()

	driver := &Driver{
		Client:        newEC2DriverTestClient(server.URL),
		Region:        "all",
		DefaultRegion: "us-east-1",
	}
	got, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("unexpected host count: %d", len(got))
	}

	hostsByID := make(map[string]string, len(got))
	regionsByID := make(map[string]string, len(got))
	for _, host := range got {
		hostsByID[host.ID] = host.HostName
		regionsByID[host.ID] = host.Region
	}
	if hostsByID["i-sg-1"] != "stack-preferred" {
		t.Fatalf("unexpected preferred hostname: %+v", got)
	}
	if hostsByID["i-sg-2"] != "" || regionsByID["i-sg-2"] != "ap-southeast-1" {
		t.Fatalf("unexpected second singapore instance: %+v", got)
	}
	if hostsByID["i-east-1"] != "east-host" || regionsByID["i-east-1"] != "ap-east-1" {
		t.Fatalf("unexpected east instance: %+v", got)
	}
}

func TestDriverGetResourceFallsBackToDefaultRegion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values := mustParseEC2BodyValues(t, r)
		if got := values.Get("Action"); got != "DescribeInstances" {
			t.Fatalf("unexpected action: %s", got)
		}
		if got := signingRegionFromAuthorization(t, r.Header.Get("Authorization")); got != "ap-southeast-1" {
			t.Fatalf("unexpected signing region: %s", got)
		}
		_, _ = w.Write([]byte(`
<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
  <reservationSet>
    <item>
      <instancesSet>
        <item>
          <instanceId>i-default</instanceId>
          <privateIpAddress>10.0.0.9</privateIpAddress>
          <instanceState><name>running</name></instanceState>
        </item>
      </instancesSet>
    </item>
  </reservationSet>
</DescribeInstancesResponse>`))
	}))
	defer server.Close()

	driver := &Driver{
		Client:        newEC2DriverTestClient(server.URL),
		DefaultRegion: "ap-southeast-1",
	}
	got, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}
	if len(got) != 1 || got[0].Region != "ap-southeast-1" || got[0].ID != "i-default" {
		t.Fatalf("unexpected hosts: %+v", got)
	}
}

func newEC2DriverTestClient(baseURL string) *api.Client {
	return api.NewClient(
		auth.New("AKID", "SECRET", ""),
		api.WithBaseURL(baseURL),
		api.WithClock(func() time.Time { return time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC) }),
		api.WithRetryPolicy(api.RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)
}

func mustParseEC2BodyValues(t *testing.T, r *http.Request) url.Values {
	t.Helper()
	values, err := url.ParseQuery(readEC2Body(t, r))
	if err != nil {
		t.Fatalf("parse request body: %v", err)
	}
	return values
}

func readEC2Body(t *testing.T, r *http.Request) string {
	t.Helper()
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(body)
}

func signingRegionFromAuthorization(t *testing.T, authorization string) string {
	t.Helper()
	const prefix = "Credential="
	start := strings.Index(authorization, prefix)
	if start < 0 {
		t.Fatalf("missing credential scope: %s", authorization)
	}
	scope := authorization[start+len(prefix):]
	if end := strings.Index(scope, ","); end >= 0 {
		scope = scope[:end]
	}
	parts := strings.Split(scope, "/")
	if len(parts) < 5 {
		t.Fatalf("invalid credential scope: %s", authorization)
	}
	return parts[2]
}
