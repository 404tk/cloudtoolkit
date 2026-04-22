package volcengine

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func TestProviderResourcesDatabaseUsesRDSDrivers(t *testing.T) {
	restoreCloudlist := setCloudlist([]string{"database"})
	defer restoreCloudlist()

	logger.SetOutput(io.Discard)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			t.Fatalf("ParseQuery() error = %v", err)
		}
		action := values.Get("Action")
		service := signedService(t, r)
		switch action {
		case "DescribeRegions":
			assertJSONBody(t, r, map[string]any{})
			switch service {
			case api.ServiceRDSMySQL:
				_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-mysql-regions"},"Result":{"Regions":[{"RegionId":"cn-beijing"},{"RegionId":"cn-guangzhou"}]}}`))
			case api.ServiceRDSPostgreSQL:
				_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-pg-regions"},"Result":{"Regions":[{"RegionId":"cn-guangzhou"}]}}`))
			case api.ServiceRDSMSSQL:
				_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-mssql-regions"},"Result":{"Regions":[{"RegionId":"cn-guangzhou"}]}}`))
			default:
				t.Fatalf("unexpected service for DescribeRegions: %s", service)
			}
		case "DescribeDBInstances":
			assertJSONBody(t, r, map[string]any{
				"PageNumber": float64(1),
				"PageSize":   float64(100),
			})
			switch service {
			case api.ServiceRDSMySQL:
				_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-mysql"},"Result":{"Instances":[{"InstanceId":"mysql-1","DBEngineVersion":"MySQL_8_0","RegionId":"cn-beijing","AddressObject":[{"NetworkType":"Private","IPAddress":"10.0.0.10","Port":"3306"},{"NetworkType":"Public","Domain":"mysql.example.com","Port":"3306"}]}],"Total":1}}`))
			case api.ServiceRDSPostgreSQL:
				_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-pg"},"Result":{"Instances":[{"InstanceId":"pg-1","DBEngineVersion":"PostgreSQL_14","RegionId":"cn-guangzhou","AddressObject":[{"NetworkType":"Private","IPAddress":"10.0.0.20","Port":"5432"}]}],"Total":1}}`))
			case api.ServiceRDSMSSQL:
				_, _ = w.Write([]byte(`{"ResponseMetadata":{"RequestId":"req-mssql"},"Result":{"InstancesInfo":[{"InstanceId":"sqlserver-1","DBEngineVersion":"2019","RegionId":"cn-guangzhou","Port":"1433","NodeDetailInfo":[{"NodeType":"Primary","NodeIP":"10.0.0.30"}]}],"Total":1}}`))
			default:
				t.Fatalf("unexpected service for DescribeDBInstances: %s", service)
			}
		default:
			t.Fatalf("unexpected action: %s", action)
		}
	}))
	defer server.Close()

	provider, err := newProvider(testOptions(map[string]string{
		utils.Provider: "volcengine",
		utils.Region:   "all",
	}), ClientConfig{
		APIOptions: testClientOptions(server.URL),
	})
	if err != nil {
		t.Fatalf("newProvider() error = %v", err)
	}

	resources, err := provider.Resources(context.Background())
	if err != nil {
		t.Fatalf("Resources() error = %v", err)
	}

	got := map[string]schema.Database{}
	for _, asset := range resources.Assets {
		db, ok := asset.(schema.Database)
		if !ok {
			t.Fatalf("unexpected asset type: %T", asset)
		}
		got[db.InstanceId] = db
	}
	if len(got) != 3 {
		t.Fatalf("unexpected database count: %d", len(got))
	}

	expect := map[string]schema.Database{
		"mysql-1": {
			Engine:        "MySQL",
			EngineVersion: "8.0",
			Region:        "cn-beijing",
			Address:       "mysql.example.com:3306",
			NetworkType:   "Public",
		},
		"pg-1": {
			Engine:        "PostgreSQL",
			EngineVersion: "14",
			Region:        "cn-guangzhou",
			Address:       "10.0.0.20:5432",
			NetworkType:   "Private",
		},
		"sqlserver-1": {
			Engine:        "SQL Server",
			EngineVersion: "2019",
			Region:        "cn-guangzhou",
			Address:       "10.0.0.30:1433",
		},
	}
	for id, want := range expect {
		db, ok := got[id]
		if !ok {
			t.Fatalf("missing database asset: %s", id)
		}
		if db.Engine != want.Engine || db.EngineVersion != want.EngineVersion || db.Region != want.Region || db.Address != want.Address || db.NetworkType != want.NetworkType {
			t.Fatalf("unexpected database asset for %s: %+v", id, db)
		}
	}
}

func TestProviderResourcesDomainUsesDNSDriver(t *testing.T) {
	restoreCloudlist := setCloudlist([]string{"domain"})
	defer restoreCloudlist()

	logger.SetOutput(io.Discard)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			t.Fatalf("ParseQuery() error = %v", err)
		}
		if service := signedService(t, r); service != "dns" {
			t.Fatalf("unexpected service: %s", service)
		}
		switch values.Get("Action") {
		case "ListZones":
			assertJSONBody(t, r, map[string]any{
				"PageNumber": float64(1),
				"PageSize":   float64(100),
			})
			_, _ = w.Write([]byte(`{"Total":2,"Zones":[{"ZoneName":"example.com","ZID":101},{"ZoneName":"example.org","ZID":102}]}`))
		case "ListRecords":
			body := decodeJSONBody(t, r)
			if body["PageNumber"] != float64(1) || body["PageSize"] != float64(100) {
				t.Fatalf("unexpected ListRecords body: %v", body)
			}
			switch body["ZID"] {
			case float64(101):
				_, _ = w.Write([]byte(`{"PageNumber":1,"PageSize":100,"TotalCount":2,"Records":[{"Host":"@","Type":"A","Value":"1.1.1.1","Enable":true},{"Host":"www","Type":"CNAME","Value":"target.example.com","Enable":false}]}`))
			case float64(102):
				_, _ = w.Write([]byte(`{"PageNumber":1,"PageSize":100,"TotalCount":1,"Records":[{"Host":"api","Type":"A","Value":"2.2.2.2","Enable":true}]}`))
			default:
				t.Fatalf("unexpected ZID: %v", body["ZID"])
			}
		default:
			t.Fatalf("unexpected action: %s", values.Get("Action"))
		}
	}))
	defer server.Close()

	provider, err := newProvider(testOptions(map[string]string{
		utils.Provider: "volcengine",
		utils.Region:   "cn-guangzhou",
	}), ClientConfig{
		APIOptions: testClientOptions(server.URL),
	})
	if err != nil {
		t.Fatalf("newProvider() error = %v", err)
	}

	resources, err := provider.Resources(context.Background())
	if err != nil {
		t.Fatalf("Resources() error = %v", err)
	}
	if len(resources.Errors) != 0 {
		t.Fatalf("unexpected resource errors: %+v", resources.Errors)
	}

	got := map[string]schema.Domain{}
	for _, asset := range resources.Assets {
		domain, ok := asset.(schema.Domain)
		if !ok {
			t.Fatalf("unexpected asset type: %T", asset)
		}
		got[domain.DomainName] = domain
	}
	if len(got) != 2 {
		t.Fatalf("unexpected domain count: %d", len(got))
	}
	if len(got["example.com"].Records) != 2 || got["example.com"].Records[1].Status != "DISABLE" {
		t.Fatalf("unexpected example.com records: %+v", got["example.com"])
	}
	if len(got["example.org"].Records) != 1 || got["example.org"].Records[0].Value != "2.2.2.2" {
		t.Fatalf("unexpected example.org records: %+v", got["example.org"])
	}
}

func testOptions(overrides map[string]string) schema.Options {
	options := schema.Options{
		utils.AccessKey: "AKID",
		utils.SecretKey: "SECRET",
	}
	for key, value := range overrides {
		options[key] = value
	}
	return options
}

func testClientOptions(baseURL string) []api.Option {
	return []api.Option{
		api.WithBaseURL(baseURL),
		api.WithClock(func() time.Time { return time.Unix(1776458501, 0).UTC() }),
		api.WithRetryPolicy(api.RetryPolicy{
			MaxAttempts: 1,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		}),
	}
}

func setCloudlist(values []string) func() {
	previous := append([]string(nil), utils.Cloudlist...)
	utils.Cloudlist = append([]string(nil), values...)
	return func() {
		utils.Cloudlist = previous
	}
}

func assertJSONBody(t *testing.T, r *http.Request, want map[string]any) {
	t.Helper()
	got := decodeJSONBody(t, r)
	if len(got) != len(want) {
		t.Fatalf("unexpected body map length: got=%v want=%v", got, want)
	}
	for key, wantValue := range want {
		if got[key] != wantValue {
			t.Fatalf("unexpected body field %s: got=%v want=%v", key, got[key], wantValue)
		}
	}
}

func decodeJSONBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v body=%s", err, string(body))
	}
	return got
}

func signedService(t *testing.T, r *http.Request) string {
	t.Helper()
	authHeader := r.Header.Get("Authorization")
	credential, found := strings.CutPrefix(authHeader, "HMAC-SHA256 Credential=")
	if !found {
		t.Fatalf("missing credential scope in authorization: %s", authHeader)
	}
	scope, _, _ := strings.Cut(credential, ",")
	parts := strings.Split(scope, "/")
	if len(parts) < 5 {
		t.Fatalf("invalid credential scope: %s", scope)
	}
	return parts[3]
}
