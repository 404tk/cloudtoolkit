package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/auth"
)

func TestEC2DescribeRegionsParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values := mustParseBodyValues(t, r)
		if got := values.Get("Action"); got != "DescribeRegions" {
			t.Fatalf("unexpected action: %s", got)
		}
		if got := values.Get("Version"); got != ec2APIVersion {
			t.Fatalf("unexpected version: %s", got)
		}
		authz := r.Header.Get("Authorization")
		if !strings.Contains(authz, "/cn-northwest-1/ec2/aws4_request") {
			t.Fatalf("unexpected authorization header: %s", authz)
		}
		_, _ = w.Write([]byte(`
<DescribeRegionsResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
  <regionInfo>
    <item><regionName> ap-southeast-1 </regionName></item>
    <item><regionName>cn-northwest-1</regionName></item>
    <item><regionName> </regionName></item>
  </regionInfo>
</DescribeRegionsResponse>`))
	}))
	defer server.Close()

	client := newEC2TestClient(server.URL)
	got, err := client.DescribeRegions(context.Background(), "cn-northwest-1")
	if err != nil {
		t.Fatalf("DescribeRegions() error = %v", err)
	}
	if len(got.Regions) != 2 {
		t.Fatalf("unexpected region count: %d", len(got.Regions))
	}
	if got.Regions[0].Name != "ap-southeast-1" || got.Regions[1].Name != "cn-northwest-1" {
		t.Fatalf("unexpected regions: %+v", got.Regions)
	}
}

func TestEC2DescribeInstancesParsesReservationsAndNextToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values := mustParseBodyValues(t, r)
		if got := values.Get("Action"); got != "DescribeInstances" {
			t.Fatalf("unexpected action: %s", got)
		}
		if got := values.Get("Version"); got != ec2APIVersion {
			t.Fatalf("unexpected version: %s", got)
		}
		if got := values.Get("NextToken"); got != "page-2" {
			t.Fatalf("unexpected next token: %s", got)
		}
		if got := values.Get("MaxResults"); got != "1000" {
			t.Fatalf("unexpected max results: %s", got)
		}
		authz := r.Header.Get("Authorization")
		if !strings.Contains(authz, "/ap-southeast-1/ec2/aws4_request") {
			t.Fatalf("unexpected authorization header: %s", authz)
		}
		_, _ = w.Write([]byte(`
<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
  <reservationSet>
    <item>
      <instancesSet>
        <item>
          <instanceId>i-001</instanceId>
          <ipAddress>1.1.1.1</ipAddress>
          <privateIpAddress>10.0.0.1</privateIpAddress>
          <dnsName>ec2-1-1-1-1.compute.amazonaws.com</dnsName>
          <instanceState><name>running</name></instanceState>
          <tagSet>
            <item><key>Name</key><value>demo-a</value></item>
          </tagSet>
        </item>
        <item>
          <instanceId>i-002</instanceId>
          <privateIpAddress>10.0.0.2</privateIpAddress>
          <instanceState><name>stopped</name></instanceState>
          <tagSet>
            <item><key>aws:cloudformation:stack-name</key><value>stack-b</value></item>
            <item><key>Name</key><value>demo-b</value></item>
          </tagSet>
        </item>
      </instancesSet>
    </item>
  </reservationSet>
  <nextToken>next-3</nextToken>
</DescribeInstancesResponse>`))
	}))
	defer server.Close()

	client := newEC2TestClient(server.URL)
	got, err := client.DescribeInstances(context.Background(), "ap-southeast-1", "page-2", 1000)
	if err != nil {
		t.Fatalf("DescribeInstances() error = %v", err)
	}
	if got.NextToken != "next-3" {
		t.Fatalf("unexpected next token: %s", got.NextToken)
	}
	if len(got.Instances) != 2 {
		t.Fatalf("unexpected instance count: %d", len(got.Instances))
	}
	if got.Instances[0].InstanceID != "i-001" || got.Instances[0].Tags[0].Value != "demo-a" {
		t.Fatalf("unexpected first instance: %+v", got.Instances[0])
	}
	if got.Instances[1].InstanceID != "i-002" || got.Instances[1].State != "stopped" || len(got.Instances[1].Tags) != 2 {
		t.Fatalf("unexpected second instance: %+v", got.Instances[1])
	}
}

func newEC2TestClient(baseURL string) *Client {
	return NewClient(
		auth.New("AKID", "SECRET", ""),
		WithBaseURL(baseURL),
		WithClock(func() time.Time { return time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC) }),
		WithRetryPolicy(RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)
}
