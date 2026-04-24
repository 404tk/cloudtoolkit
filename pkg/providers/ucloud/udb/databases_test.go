package udb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	ucloudauth "github.com/404tk/cloudtoolkit/pkg/providers/ucloud/auth"
)

func TestDriverGetDatabasesMapsClassTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		if got := r.Form.Get("Action"); got != "DescribeUDBInstance" {
			t.Fatalf("unexpected action: %s", got)
		}

		switch r.Form.Get("ClassType") {
		case "sql":
			_, _ = w.Write([]byte(`{
				"RetCode":0,
				"TotalCount":1,
				"DataSet":[{"DBId":"mysql-1","DBSubVersion":"8.0","Name":"mysql-demo","VirtualIP":"10.0.0.11","Port":3306,"VPCId":"vpc-1"}]
			}`))
		case "postgresql":
			_, _ = w.Write([]byte(`{
				"RetCode":0,
				"TotalCount":1,
				"DataSet":[{"DBId":"pg-1","DBTypeId":"postgresql-14","Name":"pg-demo","VirtualIP":"10.0.0.12","Port":5432,"SubnetId":"subnet-1"}]
			}`))
		case "nosql":
			_, _ = w.Write([]byte(`{
				"RetCode":0,
				"TotalCount":1,
				"DataSet":[{"DBId":"mongo-1","DBTypeId":"mongodb-4.4","Name":"mongo-demo","VirtualIP":"10.0.0.13","Port":27017}]
			}`))
		default:
			t.Fatalf("unexpected class type: %s", r.Form.Get("ClassType"))
		}
	}))
	defer server.Close()

	driver := &Driver{
		Credential: ucloudauth.New("ak", "sk", ""),
		Client: api.NewClient(
			ucloudauth.New("ak", "sk", ""),
			api.WithBaseURL(server.URL),
			api.WithProjectID("org-test"),
		),
		ProjectID: "org-test",
		Regions:   []string{"cn-bj2"},
	}

	got, err := driver.GetDatabases(context.Background())
	if err != nil {
		t.Fatalf("GetDatabases() error = %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len(GetDatabases()) = %d, want 3", len(got))
	}

	if got[0].InstanceId != "mysql-1" || got[0].Engine != "MySQL" || got[0].EngineVersion != "8.0" || got[0].Address != "10.0.0.11:3306" || got[0].NetworkType != "VPC" {
		t.Fatalf("unexpected mysql database: %+v", got[0])
	}
	if got[1].InstanceId != "pg-1" || got[1].Engine != "PostgreSQL" || got[1].EngineVersion != "postgresql-14" || got[1].NetworkType != "Private" {
		t.Fatalf("unexpected postgresql database: %+v", got[1])
	}
	if got[2].InstanceId != "mongo-1" || got[2].Engine != "MongoDB" || got[2].EngineVersion != "mongodb-4.4" || got[2].NetworkType != "" {
		t.Fatalf("unexpected mongodb database: %+v", got[2])
	}
}
