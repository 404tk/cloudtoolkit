package tencent

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/tat"
	"github.com/404tk/cloudtoolkit/pkg/runtime/env"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func TestNewCloudlistLogsCallerIdentity(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger.SetOutput(buffer)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-TC-Action"); got != "GetCallerIdentity" {
			t.Fatalf("unexpected action: %s", got)
		}
		_, _ = w.Write([]byte(`{"Response":{"Arn":"qcs::cam::uin/10001:uin/10001","Type":"root","UserId":"10001","RequestId":"req-sts"}}`))
	}))
	defer server.Close()

	provider, err := newProvider(testOptions(map[string]string{
		utils.Provider: "tencent",
		utils.Payload:  "cloudlist",
	}), testClientOptions(server.URL)...)
	if err != nil {
		t.Fatalf("newProvider() error = %v", err)
	}
	if provider == nil {
		t.Fatal("newProvider() returned nil provider")
	}
	if got := buffer.String(); !strings.Contains(got, "Current account ARN: qcs::cam::uin/10001:uin/10001") {
		t.Fatalf("unexpected logger output: %s", got)
	}
}

func TestProviderResourcesAccountUsesIAMDriver(t *testing.T) {
	setCloudlist(t, []string{"account"})

	next := env.Active().Clone()
	next.ListPolicies = false
	env.SetActiveForTest(t, next)

	logger.SetOutput(io.Discard)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-TC-Action"); got != "ListUsers" {
			t.Fatalf("unexpected action: %s", got)
		}
		_, _ = w.Write([]byte(`{"Response":{"Data":[{"Uin":123456,"Name":"alice","ConsoleLogin":1,"CreateTime":"2026-04-18T10:00:00Z"}],"RequestId":"req-cam"}}`))
	}))
	defer server.Close()

	provider, err := newProvider(testOptions(map[string]string{
		utils.Provider: "tencent",
	}), testClientOptions(server.URL)...)
	if err != nil {
		t.Fatalf("newProvider() error = %v", err)
	}

	resources, err := provider.Resources(context.Background())
	if err != nil {
		t.Fatalf("Resources() error = %v", err)
	}
	if len(resources.Assets) != 1 {
		t.Fatalf("unexpected asset count: %d", len(resources.Assets))
	}
	user, ok := resources.Assets[0].(schema.User)
	if !ok {
		t.Fatalf("unexpected asset type: %T", resources.Assets[0])
	}
	if user.UserName != "alice" || user.UserId != "123456" || !user.EnableLogin {
		t.Fatalf("unexpected user asset: %+v", user)
	}
}

func TestProviderResourcesDatabaseUsesAllDrivers(t *testing.T) {
	setCloudlist(t, []string{"database"})

	logger.SetOutput(io.Discard)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch action := r.Header.Get("X-TC-Action"); action {
		case "DescribeCdbZoneConfig":
			_, _ = w.Write([]byte(`{"Response":{"DataResult":{"Regions":[{"Region":"ap-guangzhou"}]},"RequestId":"req-cdb-region"}}`))
		case "DescribeSaleInfo":
			_, _ = w.Write([]byte(`{"Response":{"RegionList":[{"Region":"ap-guangzhou"}],"RequestId":"req-mariadb-region"}}`))
		case "DescribeRegions":
			switch signedService(t, r) {
			case "postgres":
				_, _ = w.Write([]byte(`{"Response":{"RegionSet":[{"Region":"ap-guangzhou","RegionState":"AVAILABLE"}],"RequestId":"req-postgres-region"}}`))
			case "sqlserver":
				_, _ = w.Write([]byte(`{"Response":{"RegionSet":[{"Region":"ap-guangzhou","RegionState":"AVAILABLE"}],"RequestId":"req-sqlserver-region"}}`))
			default:
				t.Fatalf("unexpected service for DescribeRegions: %s", signedService(t, r))
			}
		case "DescribeDBInstances":
			switch signedService(t, r) {
			case "cdb":
				_, _ = w.Write([]byte(`{"Response":{"Items":[{"InstanceId":"mysql-1","EngineVersion":"8.0","Region":"ap-guangzhou","WanStatus":1,"WanDomain":"mysql.example.com","WanPort":3306}],"RequestId":"req-cdb"}}`))
			case "mariadb":
				_, _ = w.Write([]byte(`{"Response":{"Instances":[{"InstanceId":"mariadb-1","DbVersion":"10.6","Region":"ap-guangzhou","WanStatus":0,"Vip":"10.0.0.10","Vport":3306}],"RequestId":"req-mariadb"}}`))
			case "postgres":
				_, _ = w.Write([]byte(`{"Response":{"DBInstanceSet":[{"DBInstanceId":"postgres-1","DBEngine":"PostgreSQL","DBInstanceVersion":"14","Region":"ap-guangzhou","DBInstanceNetInfo":[{"NetType":"private","Ip":"10.0.0.20","Port":5432},{"NetType":"public","Status":"opened","Address":"postgres.example.com","Port":5432}]}],"RequestId":"req-postgres"}}`))
			case "sqlserver":
				_, _ = w.Write([]byte(`{"Response":{"DBInstances":[{"InstanceId":"sqlserver-1","VersionName":"SQL Server","Version":"2019","Region":"ap-guangzhou","DnsPodDomain":"sqlserver.example.com","TgwWanVPort":1433}],"RequestId":"req-sqlserver"}}`))
			default:
				t.Fatalf("unexpected service for DescribeDBInstances: %s", signedService(t, r))
			}
		default:
			t.Fatalf("unexpected action: %s", action)
		}
	}))
	defer server.Close()

	provider, err := newProvider(testOptions(map[string]string{
		utils.Provider: "tencent",
		utils.Region:   "all",
	}), testClientOptions(server.URL)...)
	if err != nil {
		t.Fatalf("newProvider() error = %v", err)
	}

	var resources schema.Resources
	captured := captureStdout(t, func() {
		resources, err = provider.Resources(context.Background())
	})
	if err != nil {
		t.Fatalf("Resources() error = %v", err)
	}
	if captured == "" {
		t.Log("database resource enumeration produced no progress output")
	}

	got := map[string]schema.Database{}
	for _, asset := range resources.Assets {
		db, ok := asset.(schema.Database)
		if !ok {
			t.Fatalf("unexpected asset type: %T", asset)
		}
		got[db.InstanceId] = db
	}
	if len(got) != 4 {
		t.Fatalf("unexpected database count: %d", len(got))
	}

	expect := map[string]schema.Database{
		"mysql-1":     {Engine: "MySQL", EngineVersion: "8.0", Region: "ap-guangzhou", Address: "mysql.example.com:3306"},
		"mariadb-1":   {Engine: "MariaDB", EngineVersion: "10.6", Region: "ap-guangzhou", Address: "10.0.0.10:3306"},
		"postgres-1":  {Engine: "PostgreSQL", EngineVersion: "14", Region: "ap-guangzhou", Address: "postgres.example.com:5432"},
		"sqlserver-1": {Engine: "SQL Server", EngineVersion: "2019", Region: "ap-guangzhou", Address: "sqlserver.example.com:1433"},
	}
	for id, want := range expect {
		db, ok := got[id]
		if !ok {
			t.Fatalf("missing database asset: %s", id)
		}
		if db.Engine != want.Engine || db.EngineVersion != want.EngineVersion || db.Region != want.Region || db.Address != want.Address {
			t.Fatalf("unexpected database asset for %s: %+v", id, db)
		}
	}
}

func TestProviderExecuteCloudVMCommandUsesTATDriver(t *testing.T) {
	logger.SetOutput(io.Discard)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})

	tat.SetCacheHostList([]schema.Host{{
		ID:     "ins-1",
		Region: "ap-guangzhou",
		OSType: "LINUX_UNIX",
	}})
	t.Cleanup(func() {
		tat.SetCacheHostList(nil)
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-TC-Action") {
		case "RunCommand":
			body := readBody(t, r)
			if !strings.Contains(body, base64.StdEncoding.EncodeToString([]byte("echo hello"))) {
				t.Fatalf("unexpected RunCommand body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"CommandId":"cmd-1","InvocationId":"inv-1","RequestId":"req-run"}}`))
		case "DescribeInvocations":
			_, _ = w.Write([]byte(`{"Response":{"InvocationSet":[{"InvocationId":"inv-1","InvocationTaskBasicInfoSet":[{"InvocationTaskId":"task-1","TaskStatus":"RUNNING","InstanceId":"ins-1"}]}],"RequestId":"req-inv"}}`))
		case "DescribeInvocationTasks":
			_, _ = fmt.Fprintf(w, `{"Response":{"InvocationTaskSet":[{"InvocationId":"inv-1","InvocationTaskId":"task-1","TaskStatus":"SUCCESS","InstanceId":"ins-1","TaskResult":{"Output":%q}}],"RequestId":"req-task"}}`, base64.StdEncoding.EncodeToString([]byte("ok\n")))
		default:
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
	}))
	defer server.Close()

	provider, err := newProvider(testOptions(map[string]string{
		utils.Provider: "tencent",
	}), testClientOptions(server.URL)...)
	if err != nil {
		t.Fatalf("newProvider() error = %v", err)
	}

	output := captureStdout(t, func() {
		provider.ExecuteCloudVMCommand("ins-1", base64.StdEncoding.EncodeToString([]byte("echo hello")))
	})
	if !strings.Contains(output, "ok") {
		t.Fatalf("unexpected exec output: %q", output)
	}
}

func testOptions(overrides map[string]string) schema.Options {
	options := schema.Options{
		utils.AccessKey: "ak",
		utils.SecretKey: "sk",
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

func setCloudlist(t *testing.T, values []string) {
	t.Helper()
	next := env.Active().Clone()
	next.Cloudlist = append([]string(nil), values...)
	env.SetActiveForTest(t, next)
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(): %v", err)
	}
	defer func() {
		_ = writer.Close()
		os.Stdout = original
		_ = reader.Close()
	}()
	os.Stdout = writer

	done := make(chan string, 1)
	go func() {
		var buffer bytes.Buffer
		_, _ = io.Copy(&buffer, reader)
		done <- buffer.String()
	}()

	fn()
	_ = writer.Close()
	return <-done
}

func signedService(t *testing.T, r *http.Request) string {
	t.Helper()
	authHeader := r.Header.Get("Authorization")
	credential, found := strings.CutPrefix(authHeader, "TC3-HMAC-SHA256 Credential=")
	if !found {
		t.Fatalf("missing credential scope in authorization: %s", authHeader)
	}
	scope, _, _ := strings.Cut(credential, ",")
	parts := strings.Split(scope, "/")
	if len(parts) < 3 {
		t.Fatalf("invalid credential scope: %s", scope)
	}
	return parts[2]
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
