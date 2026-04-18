package cdb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListMariaDBMapsAddressesAndIgnoresFailedRegion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-TC-Action") {
		case "DescribeSaleInfo":
			if body := readBody(t, r); body != "{}" {
				t.Fatalf("unexpected DescribeSaleInfo body: %s", body)
			}
			_, _ = w.Write([]byte(`{"Response":{"RegionList":[{"Region":"ap-guangzhou"},{"Region":"ap-shanghai"}],"RequestId":"req-sale-info"}}`))
		case "DescribeDBInstances":
			switch r.Header.Get("X-TC-Region") {
			case "ap-guangzhou":
				_, _ = w.Write([]byte(`{"Response":{"Instances":[{"InstanceId":"tdsql-public","DbVersion":"10.6","Region":"ap-guangzhou","WanStatus":1,"WanDomain":"mariadb.example.com","WanPort":3306},{"InstanceId":"tdsql-private","DbVersion":"10.3","Region":"ap-guangzhou","WanStatus":0,"Vip":"10.0.2.8","Vport":3306}],"RequestId":"req-db"}}`))
			case "ap-shanghai":
				_, _ = w.Write([]byte(`{"Response":{"Error":{"Code":"InternalError","Message":"temporary failure"},"RequestId":"req-error"}}`))
			default:
				t.Fatalf("unexpected DescribeDBInstances region: %s", r.Header.Get("X-TC-Region"))
			}
		default:
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
	}))
	defer server.Close()

	driver := newTestDriver(server.URL, "all")
	databases, err := driver.ListMariaDB(context.Background())
	if err != nil {
		t.Fatalf("ListMariaDB() error = %v", err)
	}
	if len(databases) != 2 {
		t.Fatalf("unexpected database count: %d", len(databases))
	}

	byID := map[string]string{
		"tdsql-public":  "mariadb.example.com:3306",
		"tdsql-private": "10.0.2.8:3306",
	}
	for _, db := range databases {
		expect, ok := byID[db.InstanceId]
		if !ok {
			t.Fatalf("unexpected database: %+v", db)
		}
		if db.Engine != "MariaDB" || db.Address != expect {
			t.Fatalf("unexpected mapped database: %+v", db)
		}
	}
}
