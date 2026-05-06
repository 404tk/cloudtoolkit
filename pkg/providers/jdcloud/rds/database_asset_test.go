package rds

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

const sampleRDSInstances = `{"requestId":"r1","result":{"dbInstances":[
  {"instanceId":"mysql-prod","instanceName":"prod","engine":"mysql","engineVersion":"8.0","regionId":"cn-north-1",
   "instanceStatus":"RUNNING","publicDomainName":"mysql-prod-pub.jcloud-mysql.com","publicPort":3306,
   "internalDomainName":"mysql-prod.jcloud-mysql.com","internalPort":3306},
  {"instanceId":"pg-stage","instanceName":"stage","engine":"postgres","engineVersion":"15.4","regionId":"cn-north-1",
   "instanceStatus":"RUNNING","internalDomainName":"pg-stage.jcloud-pg.com","internalPort":5432}
],"totalCount":2}}`

func TestGetDatabasesListsRDSInstances(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/instances") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(sampleRDSInstances))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-north-1"}
	dbs, err := driver.GetDatabases(context.Background())
	if err != nil {
		t.Fatalf("GetDatabases: %v", err)
	}
	if len(dbs) != 2 {
		t.Fatalf("expected 2 instances, got %d", len(dbs))
	}
	if dbs[0].InstanceId != "mysql-prod" || dbs[0].Engine != "mysql" {
		t.Errorf("unexpected first db: %+v", dbs[0])
	}
	if dbs[0].NetworkType != "Public" || dbs[1].NetworkType != "Private" {
		t.Errorf("network types mismatch: %s / %s", dbs[0].NetworkType, dbs[1].NetworkType)
	}
	if !strings.Contains(dbs[0].Address, "mysql-prod-pub.jcloud-mysql.com:3306") {
		t.Errorf("expected public address in mixed form, got %q", dbs[0].Address)
	}
}

func TestGetDatabasesPropagatesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"requestId":"r-err","error":{"code":"AccessDenied","message":"forbidden"}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-north-1"}
	_, err := driver.GetDatabases(context.Background())
	if err == nil {
		t.Fatal("expected error when DescribeRDSInstances fails")
	}
	if !strings.Contains(err.Error(), "AccessDenied") {
		t.Errorf("expected AccessDenied in err, got %v", err)
	}
}

func TestGetDatabasesHandlesEmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"requestId":"r2","result":{"dbInstances":[],"totalCount":0}}`))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-north-1"}
	dbs, err := driver.GetDatabases(context.Background())
	if err != nil {
		t.Fatalf("GetDatabases: %v", err)
	}
	if len(dbs) != 0 {
		t.Errorf("expected 0 instances, got %d", len(dbs))
	}
}

// TestGetDatabasesPaginatesUsingPageNumberPageSize verifies that the driver
// walks every page when the server returns a full page on the first call.
// Regression for the previously-silent bug where pageNumber/pageSize were
// accepted by the client method but never written into the request query —
// the driver would loop on page 1 (or break early when len < pageSize).
func TestGetDatabasesPaginatesUsingPageNumberPageSize(t *testing.T) {
	pages := map[string]string{
		"1": pageBody("inst-1-", listPageSize),
		"2": pageBody("inst-2-", listPageSize),
		"3": pageBody("inst-3-", 17),
	}
	seen := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("pageNumber")
		size := r.URL.Query().Get("pageSize")
		if size != "100" {
			t.Errorf("pageSize=100 expected, got %q", size)
		}
		seen[page]++
		body, ok := pages[page]
		if !ok {
			t.Fatalf("unexpected page request: %s", page)
		}
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	driver := &Driver{Client: newTestClient(server.URL), Region: "cn-north-1"}
	dbs, err := driver.GetDatabases(context.Background())
	if err != nil {
		t.Fatalf("GetDatabases: %v", err)
	}
	if got, want := len(dbs), 2*listPageSize+17; got != want {
		t.Errorf("instance count: got %d want %d", got, want)
	}
	for _, page := range []string{"1", "2", "3"} {
		if seen[page] != 1 {
			t.Errorf("page %s seen %d times, want 1", page, seen[page])
		}
	}
}

func pageBody(prefix string, count int) string {
	out := `{"requestId":"r","result":{"dbInstances":[`
	for i := 0; i < count; i++ {
		if i > 0 {
			out += ","
		}
		out += `{"instanceId":"` + prefix + strconv.Itoa(i) + `","instanceName":"x","engine":"mysql","engineVersion":"8.0","regionId":"cn-north-1","instanceStatus":"RUNNING","internalDomainName":"d","internalPort":3306}`
	}
	out += `],"totalCount":` + strconv.Itoa(count) + `}}`
	return out
}
