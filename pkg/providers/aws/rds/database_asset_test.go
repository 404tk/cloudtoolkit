package rds

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/aws/auth"
)

func newDescribeTestDriver(baseURL string) *Driver {
	return &Driver{
		Client: api.NewClient(
			auth.New("AKID", "SECRET", ""),
			api.WithBaseURL(baseURL),
			api.WithClock(func() time.Time { return time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC) }),
			api.WithRetryPolicy(api.RetryPolicy{
				MaxAttempts: 1,
				Sleep:       func(context.Context, time.Duration) error { return nil },
			}),
		),
		Region:        "us-east-1",
		DefaultRegion: "us-east-1",
	}
}

const sampleDescribeDBInstances = `<DescribeDBInstancesResponse xmlns="https://rds.amazonaws.com/doc/2014-10-31/">
  <DescribeDBInstancesResult>
    <DBInstances>
      <DBInstance>
        <DBInstanceIdentifier>ctk-prod-db</DBInstanceIdentifier>
        <Engine>mysql</Engine>
        <EngineVersion>8.0.34</EngineVersion>
        <DBName>app</DBName>
        <DBInstanceStatus>available</DBInstanceStatus>
        <PubliclyAccessible>true</PubliclyAccessible>
        <Endpoint>
          <Address>ctk-prod-db.abc.us-east-1.rds.amazonaws.com</Address>
          <Port>3306</Port>
        </Endpoint>
        <AvailabilityZone>us-east-1a</AvailabilityZone>
      </DBInstance>
      <DBInstance>
        <DBInstanceIdentifier>ctk-internal-db</DBInstanceIdentifier>
        <Engine>postgres</Engine>
        <EngineVersion>15.4</EngineVersion>
        <DBInstanceStatus>available</DBInstanceStatus>
        <PubliclyAccessible>false</PubliclyAccessible>
        <Endpoint>
          <Address>ctk-internal-db.abc.us-east-1.rds.amazonaws.com</Address>
          <Port>5432</Port>
        </Endpoint>
        <AvailabilityZone>us-east-1b</AvailabilityZone>
      </DBInstance>
    </DBInstances>
    <Marker></Marker>
  </DescribeDBInstancesResult>
  <ResponseMetadata>
    <RequestId>req-list-rds</RequestId>
  </ResponseMetadata>
</DescribeDBInstancesResponse>`

func TestGetDatabasesParsesAndMapsFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if got := r.PostForm.Get("Action"); got != "DescribeDBInstances" {
			t.Fatalf("unexpected action: %s", got)
		}
		_, _ = w.Write([]byte(sampleDescribeDBInstances))
	}))
	defer server.Close()

	driver := newDescribeTestDriver(server.URL)
	dbs, err := driver.GetDatabases(context.Background())
	if err != nil {
		t.Fatalf("GetDatabases: %v", err)
	}
	if len(dbs) != 2 {
		t.Fatalf("expected 2 instances, got %d", len(dbs))
	}
	if dbs[0].InstanceId != "ctk-prod-db" || dbs[0].Engine != "mysql" {
		t.Errorf("unexpected first db: %+v", dbs[0])
	}
	if dbs[0].NetworkType != "Public" || dbs[1].NetworkType != "Private" {
		t.Errorf("network types mismatch: %+v / %+v", dbs[0].NetworkType, dbs[1].NetworkType)
	}
	if dbs[0].Address != "ctk-prod-db.abc.us-east-1.rds.amazonaws.com:3306" {
		t.Errorf("expected host:port address, got %q", dbs[0].Address)
	}
}

func TestGetDatabasesPropagatesAccessDeniedAcrossRegions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `<ErrorResponse><Error><Code>AccessDenied</Code><Message>denied</Message></Error></ErrorResponse>`, http.StatusForbidden)
	}))
	defer server.Close()

	driver := newDescribeTestDriver(server.URL)
	driver.Region = "all"
	_, err := driver.GetDatabases(context.Background())
	if err == nil {
		t.Fatal("expected error in 'all' mode when first region returns AccessDenied")
	}
	if !strings.Contains(err.Error(), "AccessDenied") {
		t.Errorf("expected AccessDenied in err, got %v", err)
	}
}

func TestGetDatabasesHandlesEmptyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<DescribeDBInstancesResponse><DescribeDBInstancesResult><DBInstances/><Marker/></DescribeDBInstancesResult><ResponseMetadata><RequestId>r2</RequestId></ResponseMetadata></DescribeDBInstancesResponse>`))
	}))
	defer server.Close()

	driver := newDescribeTestDriver(server.URL)
	dbs, err := driver.GetDatabases(context.Background())
	if err != nil {
		t.Fatalf("GetDatabases: %v", err)
	}
	if len(dbs) != 0 {
		t.Errorf("expected 0 instances, got %d", len(dbs))
	}
}
