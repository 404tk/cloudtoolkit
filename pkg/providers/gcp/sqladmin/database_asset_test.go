package sqladmin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const sampleSQLInstances = `{"items":[
  {"name":"ctk-prod-mysql","databaseVersion":"MYSQL_8_0","region":"us-central1","state":"RUNNABLE",
   "ipAddresses":[{"type":"PRIMARY","ipAddress":"35.232.0.10"},{"type":"OUTGOING","ipAddress":"34.122.0.5"}],
   "connectionName":"proj-1:us-central1:ctk-prod-mysql","backendType":"SECOND_GEN","instanceType":"CLOUD_SQL_INSTANCE",
   "settings":{"tier":"db-n1-standard-2","ipConfiguration":{"ipv4Enabled":true}}},
  {"name":"ctk-internal-pg","databaseVersion":"POSTGRES_14","region":"us-east1","state":"RUNNABLE",
   "ipAddresses":[{"type":"PRIVATE","ipAddress":"10.0.0.20"}],
   "connectionName":"proj-1:us-east1:ctk-internal-pg","backendType":"SECOND_GEN","instanceType":"CLOUD_SQL_INSTANCE",
   "settings":{"tier":"db-custom-2-8192","ipConfiguration":{"ipv4Enabled":false,"privateNetwork":"projects/proj-1/global/networks/default"}}}
]}`

func TestGetDatabasesListsCloudSQLInstances(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_, _ = w.Write([]byte(`{"access_token":"demo","token_type":"Bearer","expires_in":3600}`))
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/sql/v1beta4/projects/proj-1/instances") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(sampleSQLInstances))
	}))
	defer server.Close()

	driver := &Driver{Client: newDriverClient(t, server), Projects: []string{"proj-1"}}
	dbs, err := driver.GetDatabases(context.Background())
	if err != nil {
		t.Fatalf("GetDatabases: %v", err)
	}
	if len(dbs) != 2 {
		t.Fatalf("expected 2 instances, got %d", len(dbs))
	}
	if dbs[0].Engine != "mysql" || dbs[0].EngineVersion != "MYSQL_8_0" {
		t.Errorf("engine mapping mismatch: %+v", dbs[0])
	}
	if dbs[0].Address != "35.232.0.10" {
		t.Errorf("expected PRIMARY ip, got %q", dbs[0].Address)
	}
	if dbs[0].NetworkType != "Public" || dbs[1].NetworkType != "Private" {
		t.Errorf("network types mismatch: %s / %s", dbs[0].NetworkType, dbs[1].NetworkType)
	}
}

func TestGetDatabasesPropagatesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_, _ = w.Write([]byte(`{"access_token":"demo","token_type":"Bearer","expires_in":3600}`))
			return
		}
		http.Error(w, `{"error":{"code":403,"message":"forbidden","status":"PERMISSION_DENIED"}}`, http.StatusForbidden)
	}))
	defer server.Close()

	driver := &Driver{Client: newDriverClient(t, server), Projects: []string{"proj-1"}}
	_, err := driver.GetDatabases(context.Background())
	if err == nil {
		t.Fatal("expected error when instances.list fails")
	}
	if !strings.Contains(err.Error(), "PERMISSION_DENIED") {
		t.Errorf("expected PERMISSION_DENIED in err, got %v", err)
	}
}

func TestGetDatabasesHandlesEmptyProjectList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_, _ = w.Write([]byte(`{"access_token":"demo","token_type":"Bearer","expires_in":3600}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newDriverClient(t, server), Projects: []string{"proj-1"}}
	dbs, err := driver.GetDatabases(context.Background())
	if err != nil {
		t.Fatalf("GetDatabases: %v", err)
	}
	if len(dbs) != 0 {
		t.Errorf("expected 0 instances, got %d", len(dbs))
	}
}
