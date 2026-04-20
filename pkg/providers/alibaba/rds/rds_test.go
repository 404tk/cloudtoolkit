package rds

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

func TestGetDatabasesAllRegionsWithPaginationAndCache(t *testing.T) {
	logger.SetOutput(io.Discard)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})
	SetCacheDBList(nil)
	t.Cleanup(func() {
		SetCacheDBList(nil)
	})

	var (
		mu    sync.Mutex
		calls = make(map[string]int)
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("Action")
		switch action {
		case "DescribeRegions":
			mu.Lock()
			calls["DescribeRegions"]++
			mu.Unlock()
			_, _ = io.WriteString(w, `{"RequestId":"req-regions","Regions":{"RDSRegion":[{"RegionId":"cn-hangzhou"},{"RegionId":"cn-hangzhou"},{"RegionId":"cn-shanghai"}]}}`)
		case "DescribeDBInstances":
			region := r.URL.Query().Get("RegionId")
			page := r.URL.Query().Get("PageNumber")
			key := action + ":" + region + ":" + page
			mu.Lock()
			calls[key]++
			mu.Unlock()
			switch key {
			case "DescribeDBInstances:cn-hangzhou:1":
				_, _ = io.WriteString(w, `{"RequestId":"req-hz-1","PageNumber":1,"PageRecordCount":1,"TotalRecordCount":2,"Items":{"DBInstance":[{"DBInstanceId":"rm-hz-1","Engine":"MySQL","EngineVersion":"8.0","RegionId":"cn-hangzhou","ConnectionString":"hz-1.mysql.rds.aliyuncs.com","InstanceNetworkType":"VPC"}]}}`)
			case "DescribeDBInstances:cn-hangzhou:2":
				_, _ = io.WriteString(w, `{"RequestId":"req-hz-2","PageNumber":2,"PageRecordCount":1,"TotalRecordCount":2,"Items":{"DBInstance":[{"DBInstanceId":"rm-hz-2","Engine":"PostgreSQL","EngineVersion":"14.0","RegionId":"cn-hangzhou","ConnectionString":"hz-2.pg.rds.aliyuncs.com","InstanceNetworkType":"Classic"}]}}`)
			case "DescribeDBInstances:cn-shanghai:1":
				_, _ = io.WriteString(w, `{"RequestId":"req-sh-1","PageNumber":1,"PageRecordCount":1,"TotalRecordCount":1,"Items":{"DBInstance":[{"DBInstanceId":"rm-sh-1","Engine":"SQLServer","EngineVersion":"2019","RegionId":"cn-shanghai","ConnectionString":"sh-1.sqlserver.rds.aliyuncs.com","InstanceNetworkType":"VPC"}]}}`)
			default:
				t.Fatalf("unexpected describe db instances request: %s", key)
			}
		case "DescribeDatabases":
			instanceID := r.URL.Query().Get("DBInstanceId")
			key := action + ":" + instanceID
			mu.Lock()
			calls[key]++
			mu.Unlock()
			switch instanceID {
			case "rm-hz-1":
				_, _ = io.WriteString(w, `{"RequestId":"req-dbs-hz-1","Databases":{"Database":[{"DBName":"app"},{"DBName":"metrics"}]}}`)
			case "rm-hz-2":
				_, _ = io.WriteString(w, `{"RequestId":"req-dbs-hz-2","Databases":{"Database":[{"DBName":"analytics"}]}}`)
			case "rm-sh-1":
				_, _ = io.WriteString(w, `{"RequestId":"req-dbs-sh-1","Databases":{"Database":[]}}`)
			default:
				t.Fatalf("unexpected describe databases request: %s", instanceID)
			}
		default:
			t.Fatalf("unexpected action: %s", action)
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	databases, err := driver.GetDatabases(context.Background())
	if err != nil {
		t.Fatalf("GetDatabases() error = %v", err)
	}
	if len(databases) != 3 {
		t.Fatalf("unexpected database count: %d", len(databases))
	}

	if calls["DescribeRegions"] != 1 {
		t.Fatalf("unexpected DescribeRegions count: %v", calls)
	}
	if calls["DescribeDBInstances:cn-hangzhou:1"] != 1 || calls["DescribeDBInstances:cn-hangzhou:2"] != 1 || calls["DescribeDBInstances:cn-shanghai:1"] != 1 {
		t.Fatalf("unexpected DescribeDBInstances calls: %v", calls)
	}
	if calls["DescribeDatabases:rm-hz-1"] != 1 || calls["DescribeDatabases:rm-hz-2"] != 1 || calls["DescribeDatabases:rm-sh-1"] != 1 {
		t.Fatalf("unexpected DescribeDatabases calls: %v", calls)
	}

	assertDatabase(t, databases, schema.Database{
		InstanceId:    "rm-hz-1",
		Engine:        "MySQL",
		EngineVersion: "8.0",
		Region:        "cn-hangzhou",
		Address:       "hz-1.mysql.rds.aliyuncs.com",
		NetworkType:   "VPC",
		DBNames:       "app,metrics",
	})
	assertDatabase(t, databases, schema.Database{
		InstanceId:    "rm-hz-2",
		Engine:        "PostgreSQL",
		EngineVersion: "14.0",
		Region:        "cn-hangzhou",
		Address:       "hz-2.pg.rds.aliyuncs.com",
		NetworkType:   "Classic",
		DBNames:       "analytics",
	})
	assertDatabase(t, databases, schema.Database{
		InstanceId:    "rm-sh-1",
		Engine:        "SQLServer",
		EngineVersion: "2019",
		Region:        "cn-shanghai",
		Address:       "sh-1.sqlserver.rds.aliyuncs.com",
		NetworkType:   "VPC",
		DBNames:       "",
	})

	cached := GetCacheDBList()
	if len(cached) != len(databases) {
		t.Fatalf("unexpected cache count: %d", len(cached))
	}
	assertDatabase(t, cached, schema.Database{
		InstanceId:    "rm-hz-1",
		Engine:        "MySQL",
		EngineVersion: "8.0",
		Region:        "cn-hangzhou",
		Address:       "hz-1.mysql.rds.aliyuncs.com",
		NetworkType:   "VPC",
		DBNames:       "app,metrics",
	})
}

func TestCreateAccountCreatesReadonlyUserAndGrantsPrivilege(t *testing.T) {
	logger.SetOutput(io.Discard)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})

	originalAccount := utils.RDSAccount
	utils.RDSAccount = "readonly:Secret!1"
	t.Cleanup(func() {
		utils.RDSAccount = originalAccount
	})

	var actions []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("Action")
		actions = append(actions, action)
		switch action {
		case "CreateAccount":
			if got := r.URL.Query().Get("AccountName"); got != "readonly" {
				t.Fatalf("unexpected account name: %s", got)
			}
			if got := r.URL.Query().Get("AccountPassword"); got != "Secret!1" {
				t.Fatalf("unexpected account password: %s", got)
			}
			if got := r.URL.Query().Get("AccountType"); got != "Normal" {
				t.Fatalf("unexpected account type: %s", got)
			}
			_, _ = io.WriteString(w, `{"RequestId":"req-create-account"}`)
		case "GrantAccountPrivilege":
			if got := r.URL.Query().Get("DBName"); got != "appdb" {
				t.Fatalf("unexpected db name: %s", got)
			}
			if got := r.URL.Query().Get("AccountPrivilege"); got != "ReadOnly" {
				t.Fatalf("unexpected privilege: %s", got)
			}
			_, _ = io.WriteString(w, `{"RequestId":"req-grant"}`)
		default:
			t.Fatalf("unexpected action: %s", action)
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	driver.Region = "cn-hangzhou"

	var ok bool
	output := captureStdout(t, func() {
		ok = driver.CreateAccount("rm-1", "appdb")
	})
	if !ok {
		t.Fatalf("CreateAccount() returned false")
	}
	if strings.Join(actions, ",") != "CreateAccount,GrantAccountPrivilege" {
		t.Fatalf("unexpected action sequence: %v", actions)
	}
	if !strings.Contains(output, "readonly") || !strings.Contains(output, "Secret!1") || !strings.Contains(output, "ReadOnly") {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestDeleteAccountUsesConfiguredCredential(t *testing.T) {
	logger.SetOutput(io.Discard)
	t.Cleanup(func() {
		logger.SetOutput(nil)
	})

	originalAccount := utils.RDSAccount
	utils.RDSAccount = "readonly:Secret!1"
	t.Cleanup(func() {
		utils.RDSAccount = originalAccount
	})

	var actions []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("Action")
		actions = append(actions, action)
		if action != "DeleteAccount" {
			t.Fatalf("unexpected action: %s", action)
		}
		if got := r.URL.Query().Get("AccountName"); got != "readonly" {
			t.Fatalf("unexpected account name: %s", got)
		}
		if got := r.URL.Query().Get("DBInstanceId"); got != "rm-1" {
			t.Fatalf("unexpected instance id: %s", got)
		}
		_, _ = io.WriteString(w, `{"RequestId":"req-delete-account"}`)
	}))
	defer server.Close()

	driver := newTestDriver(server.URL)
	driver.Region = "cn-hangzhou"
	driver.DeleteAccount("rm-1")

	if strings.Join(actions, ",") != "DeleteAccount" {
		t.Fatalf("unexpected action sequence: %v", actions)
	}
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

func assertDatabase(t *testing.T, databases []schema.Database, want schema.Database) {
	t.Helper()

	for _, database := range databases {
		if database.InstanceId != want.InstanceId {
			continue
		}
		if database != want {
			t.Fatalf("unexpected database for %s: got %+v want %+v", want.InstanceId, database, want)
		}
		return
	}
	t.Fatalf("database %s not found in %+v", want.InstanceId, databases)
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = writer

	done := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(reader)
		done <- string(data)
	}()

	fn()

	_ = writer.Close()
	os.Stdout = originalStdout
	return <-done
}
