package ecs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/auth"
)

func TestDriverGetResourceAllRegionsAndPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			t.Fatalf("parse query: %v", err)
		}
		switch values.Get("Action") {
		case "DescribeRegions":
			_, _ = w.Write(mustJSON(t, api.DescribeRegionsResponse{
				Result: struct {
					NextToken string          `json:"NextToken"`
					Regions   []api.ECSRegion `json:"Regions"`
				}{
					Regions: []api.ECSRegion{{RegionID: "cn-beijing"}, {RegionID: "cn-shanghai"}},
				},
			}))
		case "DescribeInstances":
			region := scopeRegion(t, r.Header.Get(api.HeaderAuthorization))
			nextToken := values.Get("NextToken")
			switch {
			case region == "cn-beijing" && nextToken == "":
				_, _ = w.Write(mustJSON(t, describeInstancesWithIDs(generateIDs("bj", 100), "page-2")))
			case region == "cn-beijing" && nextToken == "page-2":
				_, _ = w.Write(mustJSON(t, describeInstancesWithIDs([]string{"bj-100"}, "")))
			case region == "cn-shanghai":
				_, _ = w.Write(mustJSON(t, describeInstancesWithIDs([]string{"sh-001"}, "")))
			default:
				t.Fatalf("unexpected region/token: %s %q", region, nextToken)
			}
		default:
			t.Fatalf("unexpected action: %s", values.Get("Action"))
		}
	}))
	defer server.Close()

	driver := &Driver{
		Client: newTestClient(server.URL),
		Region: "all",
	}
	got, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}
	if len(got) != 102 {
		t.Fatalf("unexpected host count: %d", len(got))
	}
	ids := make(map[string]string, len(got))
	for _, host := range got {
		ids[host.ID] = host.Region
	}
	if ids["bj-000"] != "cn-beijing" || ids["bj-100"] != "cn-beijing" || ids["sh-001"] != "cn-shanghai" {
		t.Fatalf("unexpected host map: %+v", ids)
	}
}

func TestDriverGetResourceUsesDefaultRegionWhenEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			t.Fatalf("parse query: %v", err)
		}
		if values.Get("Action") != "DescribeInstances" {
			t.Fatalf("unexpected action: %s", values.Get("Action"))
		}
		if got := scopeRegion(t, r.Header.Get(api.HeaderAuthorization)); got != "cn-beijing" {
			t.Fatalf("unexpected signing region: %s", got)
		}
		_, _ = w.Write(mustJSON(t, describeInstancesWithIDs([]string{"default-1"}, "")))
	}))
	defer server.Close()

	driver := &Driver{
		Client: newTestClient(server.URL),
		Region: "",
	}
	got, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}
	if len(got) != 1 || got[0].Region != "cn-beijing" {
		t.Fatalf("unexpected hosts: %+v", got)
	}
}

func TestDriverGetResourceReturnsPartialRegionErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			t.Fatalf("parse query: %v", err)
		}
		switch values.Get("Action") {
		case "DescribeRegions":
			_, _ = w.Write(mustJSON(t, api.DescribeRegionsResponse{
				Result: struct {
					NextToken string          `json:"NextToken"`
					Regions   []api.ECSRegion `json:"Regions"`
				}{
					Regions: []api.ECSRegion{{RegionID: "cn-beijing"}, {RegionID: "cn-shanghai"}},
				},
			}))
		case "DescribeInstances":
			switch region := scopeRegion(t, r.Header.Get(api.HeaderAuthorization)); region {
			case "cn-beijing":
				_, _ = w.Write(mustJSON(t, describeInstancesWithIDs([]string{"bj-001"}, "")))
			case "cn-shanghai":
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-sh","Error":{"Code":"Forbidden","Message":"denied"}}}`))
			default:
				t.Fatalf("unexpected region: %s", region)
			}
		default:
			t.Fatalf("unexpected action: %s", values.Get("Action"))
		}
	}))
	defer server.Close()

	driver := &Driver{
		Client: newTestClient(server.URL),
		Region: "all",
	}
	got, err := driver.GetResource(context.Background())
	if len(got) != 1 || got[0].ID != "bj-001" {
		t.Fatalf("unexpected hosts: %+v", got)
	}
	if err == nil {
		t.Fatal("expected partial region error")
	}
	if !strings.Contains(err.Error(), "cn-shanghai") || !strings.Contains(err.Error(), "Forbidden") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func newTestClient(baseURL string) *api.Client {
	return api.NewClient(
		auth.New("AKID", "SECRET", ""),
		api.WithBaseURL(baseURL),
		api.WithClock(func() time.Time { return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC) }),
		api.WithRetryPolicy(api.RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	)
}

func scopeRegion(t *testing.T, authz string) string {
	t.Helper()
	const prefix = "HMAC-SHA256 Credential="
	if !strings.HasPrefix(authz, prefix) {
		t.Fatalf("unexpected authorization header: %s", authz)
	}
	credential := strings.Split(strings.TrimPrefix(authz, prefix), ",")[0]
	parts := strings.Split(credential, "/")
	if len(parts) < 5 {
		t.Fatalf("unexpected credential scope: %s", credential)
	}
	return parts[2]
}

func generateIDs(prefix string, count int) []string {
	ids := make([]string, 0, count)
	for i := 0; i < count; i++ {
		ids = append(ids, prefix+"-"+leftPad3(i))
	}
	return ids
}

func leftPad3(v int) string {
	switch {
	case v < 10:
		return "00" + strconv.Itoa(v)
	case v < 100:
		return "0" + strconv.Itoa(v)
	default:
		return strconv.Itoa(v)
	}
}

func describeInstancesWithIDs(ids []string, nextToken string) api.DescribeInstancesResponse {
	resp := api.DescribeInstancesResponse{}
	resp.Result.NextToken = nextToken
	resp.Result.Instances = make([]api.ECSInstance, 0, len(ids))
	for _, id := range ids {
		resp.Result.Instances = append(resp.Result.Instances, api.ECSInstance{
			InstanceID: id,
			Hostname:   id + ".example",
			Status:     "Running",
			OSType:     "Linux",
			EipAddress: api.ECSEipAddress{IPAddress: "1.1.1.1"},
			NetworkInterfaces: []api.ECSNetworkInterface{
				{PrimaryIPAddress: "10.0.0.1"},
			},
		})
	}
	return resp
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return data
}
