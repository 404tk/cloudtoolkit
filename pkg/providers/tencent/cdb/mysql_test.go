package cdb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
)

func TestListMySQLMapsAddressesAndSkipsUnsupportedRegion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-TC-Action") {
		case "DescribeCdbZoneConfig":
			if got := r.Header.Get("X-TC-Region"); got != api.DefaultRegion {
				t.Fatalf("unexpected DescribeCdbZoneConfig region: %s", got)
			}
			if body := readBody(t, r); body != "{}" {
				t.Fatalf("unexpected DescribeCdbZoneConfig body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"DataResult":{"Regions":[{"Region":"ap-guangzhou"},{"Region":"ap-beijing"}]},"RequestId":"req-regions"}}`))
		case "DescribeDBInstances":
			switch r.Header.Get("X-TC-Region") {
			case "ap-guangzhou":
				if body := readBody(t, r); body != "{}" {
					t.Fatalf("unexpected DescribeDBInstances body: %s", body)
				}
				_, _ = w.Write([]byte(`{"Response":{"Items":[{"InstanceId":"cdb-public","EngineVersion":"8.0","Region":"ap-guangzhou","WanStatus":1,"WanDomain":"mysql.example.com","WanPort":3306},{"InstanceId":"cdb-private","EngineVersion":"5.7","Region":"ap-guangzhou","WanStatus":0,"Vip":"10.0.0.8","Vport":3306}],"RequestId":"req-db"}}`))
			case "ap-beijing":
				_, _ = w.Write([]byte(`{"Response":{"Error":{"Code":"InvalidParameter.UnsupportedRegion","Message":"unsupported"},"RequestId":"req-unsupported"}}`))
			default:
				t.Fatalf("unexpected DescribeDBInstances region: %s", r.Header.Get("X-TC-Region"))
			}
		default:
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "all")
	databases, err := driver.ListMySQL(context.Background())
	if err != nil {
		t.Fatalf("ListMySQL() error = %v", err)
	}
	if len(databases) != 2 {
		t.Fatalf("unexpected database count: %d", len(databases))
	}

	byID := map[string]struct {
		Address string
		Version string
	}{
		"cdb-public":  {Address: "mysql.example.com:3306", Version: "8.0"},
		"cdb-private": {Address: "10.0.0.8:3306", Version: "5.7"},
	}
	for _, db := range databases {
		expect, ok := byID[db.InstanceId]
		if !ok {
			t.Fatalf("unexpected database: %+v", db)
		}
		if db.Engine != "MySQL" || db.Address != expect.Address || db.EngineVersion != expect.Version {
			t.Fatalf("unexpected mapped database: %+v", db)
		}
	}
}

func TestListMySQLUsesDefaultRegionWhenUnset(t *testing.T) {
	sawRegionDiscovery := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-TC-Action") {
		case "DescribeCdbZoneConfig":
			sawRegionDiscovery = true
			t.Fatal("DescribeCdbZoneConfig should not be called when region is unset")
		case "DescribeDBInstances":
			if got := r.Header.Get("X-TC-Region"); got != api.DefaultRegion {
				t.Fatalf("unexpected default region: %s", got)
			}
			_, _ = w.Write([]byte(`{"Response":{"Items":[{"InstanceId":"cdb-default","EngineVersion":"8.0","Region":"ap-guangzhou","WanStatus":0,"Vip":"10.0.1.8","Vport":3306}],"RequestId":"req-db"}}`))
		default:
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "")
	databases, err := driver.ListMySQL(context.Background())
	if err != nil {
		t.Fatalf("ListMySQL() error = %v", err)
	}
	if sawRegionDiscovery {
		t.Fatal("unexpected region discovery call")
	}
	if len(databases) != 1 || databases[0].Region != api.DefaultRegion {
		t.Fatalf("unexpected databases: %+v", databases)
	}
}
