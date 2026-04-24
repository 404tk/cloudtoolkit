package udns

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	ucloudauth "github.com/404tk/cloudtoolkit/pkg/providers/ucloud/auth"
)

func TestDriverGetDomainsListsZonesAndRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}

		switch r.Form.Get("Action") {
		case "DescribeUDNSZone":
			_, _ = w.Write([]byte(`{
				"RetCode":0,
				"TotalCount":1,
				"DNSZoneInfos":[{"DNSZoneId":"zone-1","DNSZoneName":"example.com"}]
			}`))
		case "DescribeUDNSRecord":
			if got := r.Form.Get("DNSZoneId"); got != "zone-1" {
				t.Fatalf("unexpected zone id: %s", got)
			}
			_, _ = w.Write([]byte(`{
				"RetCode":0,
				"TotalCount":1,
				"RecordInfos":[
					{
						"Name":"www",
						"Type":"A",
						"ValueSet":[{"Data":"1.2.3.4","IsEnabled":1}]
					}
				]
			}`))
		default:
			t.Fatalf("unexpected action: %s", r.Form.Get("Action"))
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

	got, err := driver.GetDomains(context.Background())
	if err != nil {
		t.Fatalf("GetDomains() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(GetDomains()) = %d, want 1", len(got))
	}
	if got[0].DomainName != "example.com" || len(got[0].Records) != 1 {
		t.Fatalf("unexpected domain: %+v", got[0])
	}
	record := got[0].Records[0]
	if record.RR != "www" || record.Type != "A" || record.Value != "1.2.3.4" || record.Status != "ENABLE" {
		t.Fatalf("unexpected record: %+v", record)
	}
}
