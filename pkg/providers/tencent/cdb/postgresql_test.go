package cdb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListPostgreSQLFiltersRegionsAndPrefersOpenedPublicAddress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-TC-Action") {
		case "DescribeRegions":
			if body := readBody(t, r); body != "{}" {
				t.Fatalf("unexpected DescribeRegions body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"RegionSet":[{"Region":"ap-guangzhou","RegionState":"AVAILABLE"},{"Region":"ap-beijing","RegionState":"UNAVAILABLE"}],"RequestId":"req-regions"}}`))
		case "DescribeDBInstances":
			if got := r.Header.Get("X-TC-Region"); got != "ap-guangzhou" {
				t.Fatalf("unexpected DescribeDBInstances region: %s", got)
			}
			_, _ = w.Write([]byte(`{"Response":{"DBInstanceSet":[{"DBInstanceId":"pg-public","DBEngine":"postgresql","DBInstanceVersion":"15.3","Region":"ap-guangzhou","DBInstanceNetInfo":[{"Ip":"10.0.3.8","Port":5432,"NetType":"private","Status":"opened"},{"Address":"pg.example.com","Port":5432,"NetType":"public","Status":"opened"}]},{"DBInstanceId":"pg-private","DBEngine":"postgresql","DBInstanceVersion":"14.9","Region":"ap-guangzhou","DBInstanceNetInfo":[{"Ip":"10.0.3.9","Port":5432,"NetType":"private","Status":"opened"},{"Address":"closed.example.com","Port":5432,"NetType":"public","Status":"closed"}]}],"RequestId":"req-db"}}`))
		default:
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "all")
	databases, err := driver.ListPostgreSQL(context.Background())
	if err != nil {
		t.Fatalf("ListPostgreSQL() error = %v", err)
	}
	if len(databases) != 2 {
		t.Fatalf("unexpected database count: %d", len(databases))
	}

	byID := map[string]string{
		"pg-public":  "pg.example.com:5432",
		"pg-private": "10.0.3.9:5432",
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
