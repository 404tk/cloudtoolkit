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
	}), testClientOptions(server.URL)...)
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
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v body=%s", err, string(body))
	}
	if len(got) != len(want) {
		t.Fatalf("unexpected body map length: got=%v want=%v", got, want)
	}
	for key, wantValue := range want {
		if got[key] != wantValue {
			t.Fatalf("unexpected body field %s: got=%v want=%v", key, got[key], wantValue)
		}
	}
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
