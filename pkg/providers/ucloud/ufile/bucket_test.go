package ufile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	ucloudauth "github.com/404tk/cloudtoolkit/pkg/providers/ucloud/auth"
)

func TestDriverGetBucketsMapsResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		if got := r.Form.Get("Action"); got != "DescribeBucket" {
			t.Fatalf("unexpected action: %s", got)
		}
		if got := r.Form.Get("Region"); got != "cn-bj2" {
			t.Fatalf("unexpected region: %s", got)
		}
		_, _ = w.Write([]byte(`{
			"RetCode":0,
			"DataSet":[
				{"BucketName":"bucket-a","Region":"cn-bj2"},
				{"BucketName":"bucket-b","Region":""}
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
		Region:    "cn-bj2",
	}

	got, err := driver.GetBuckets(context.Background())
	if err != nil {
		t.Fatalf("GetBuckets() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(GetBuckets()) = %d, want 2", len(got))
	}
	if got[0].BucketName != "bucket-a" || got[0].Region != "cn-bj2" {
		t.Fatalf("unexpected first bucket: %+v", got[0])
	}
	if got[1].BucketName != "bucket-b" || got[1].Region != "cn-bj2" {
		t.Fatalf("unexpected second bucket: %+v", got[1])
	}
}
