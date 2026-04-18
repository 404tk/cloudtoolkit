package cdb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListSQLServerFiltersRegionsAndSelectsPublicDNSWhenPresent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-TC-Action") {
		case "DescribeRegions":
			if body := readBody(t, r); body != "{}" {
				t.Fatalf("unexpected DescribeRegions body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"RegionSet":[{"Region":"ap-guangzhou","RegionState":"AVAILABLE"},{"Region":"ap-singapore","RegionState":"UNAVAILABLE"}],"RequestId":"req-regions"}}`))
		case "DescribeDBInstances":
			if got := r.Header.Get("X-TC-Region"); got != "ap-guangzhou" {
				t.Fatalf("unexpected DescribeDBInstances region: %s", got)
			}
			_, _ = w.Write([]byte(`{"Response":{"DBInstances":[{"InstanceId":"mssql-public","VersionName":"SQLServer 2019","Version":"15.0","Region":"ap-guangzhou","DnsPodDomain":"mssql.example.com","TgwWanVPort":1433,"Vip":"10.0.4.8","Vport":1433},{"InstanceId":"mssql-private","VersionName":"SQLServer 2017","Version":"14.0","Region":"ap-guangzhou","DnsPodDomain":"","Vip":"10.0.4.9","Vport":1433}],"RequestId":"req-db"}}`))
		default:
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "all")
	databases, err := driver.ListSQLServer(context.Background())
	if err != nil {
		t.Fatalf("ListSQLServer() error = %v", err)
	}
	if len(databases) != 2 {
		t.Fatalf("unexpected database count: %d", len(databases))
	}

	byID := map[string]string{
		"mssql-public":  "mssql.example.com:1433",
		"mssql-private": "10.0.4.9:1433",
	}
	for _, db := range databases {
		expect, ok := byID[db.InstanceId]
		if !ok {
			t.Fatalf("unexpected database: %+v", db)
		}
		if db.Address != expect {
			t.Fatalf("unexpected mapped database: %+v", db)
		}
	}
}
