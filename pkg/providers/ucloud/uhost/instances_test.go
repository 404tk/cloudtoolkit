package uhost

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	ucloudauth "github.com/404tk/cloudtoolkit/pkg/providers/ucloud/auth"
)

func TestDriverGetResourceMapsInstances(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		if got := r.Form.Get("Action"); got != "DescribeUHostInstance" {
			t.Fatalf("unexpected action: %s", got)
		}
		if got := r.Form.Get("Region"); got != "cn-bj2" {
			t.Fatalf("unexpected region: %s", got)
		}
		if got := r.Form.Get("ProjectId"); got != "org-test" {
			t.Fatalf("unexpected project id: %s", got)
		}

		_, _ = w.Write([]byte(`{
			"RetCode":0,
			"TotalCount":2,
			"UHostSet":[
				{
					"Name":"demo-1",
					"UHostId":"uhost-1",
					"State":"Running",
					"OsType":"Linux",
					"IPSet":[
						{"IP":"10.0.0.10","Type":"Private","Default":"true","IPMode":"IPv4","Weight":0},
						{"IP":"1.1.1.1","Type":"International","Default":"false","IPMode":"IPv4","Weight":10}
					]
				},
				{
					"Name":"demo-2",
					"UHostId":"uhost-2",
					"State":"Stopped",
					"OsType":"Windows",
					"IPSet":[
						{"IP":"10.0.0.20","Type":"Private","Default":"true","IPMode":"IPv4","Weight":0}
					]
				}
			]
		}`))
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

	got, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(GetResource()) = %d, want 2", len(got))
	}
	if got[0].ID != "uhost-1" || got[0].PublicIPv4 != "1.1.1.1" || got[0].PrivateIpv4 != "10.0.0.10" || got[0].OSType != "Linux" {
		t.Fatalf("unexpected first host: %+v", got[0])
	}
	if got[1].ID != "uhost-2" || got[1].PublicIPv4 != "" || got[1].PrivateIpv4 != "10.0.0.20" || got[1].OSType != "Windows" {
		t.Fatalf("unexpected second host: %+v", got[1])
	}
}
