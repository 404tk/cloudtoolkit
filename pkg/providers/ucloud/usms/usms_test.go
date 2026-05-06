package usms

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/auth"
)

func newDriver(baseURL string) *Driver {
	credential := auth.New("ucloudpubkey-EXAMPLE", "ucloudprivkey-EXAMPLE", "")
	return &Driver{
		Credential: credential,
		Region:     "cn-bj2",
		Client: api.NewClient(credential,
			api.WithBaseURL(baseURL),
			api.WithRetryPolicy(api.RetryPolicy{MaxAttempts: 1}),
		),
	}
}

func TestGetResourceMapsSignaturesAndTemplates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch r.Form.Get("Action") {
		case "DescribeUSMSSignature":
			_, _ = w.Write([]byte(`{"Action":"DescribeUSMSSignatureResponse","RetCode":0,"TotalCount":2,"Data":[
  {"SigId":"sig-1","SigContent":"ctk-prod","Status":0,"SigType":2,"UpdateTime":1713427200},
  {"SigId":"sig-2","SigContent":"ctk-stage","Status":1,"SigType":1,"UpdateTime":1713427260}
]}`))
		case "DescribeUSMSTemplate":
			_, _ = w.Write([]byte(`{"Action":"DescribeUSMSTemplateResponse","RetCode":0,"TotalCount":1,"Data":[
  {"TemplateId":"tpl-1","Template":"Code is {1}","TemplateName":"OTP","TemplateType":0,"Status":0,"UpdateTime":1713427300}
]}`))
		default:
			t.Fatalf("unexpected action: %s", r.Form.Get("Action"))
		}
	}))
	defer server.Close()

	driver := newDriver(server.URL)
	res, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource: %v", err)
	}
	if len(res.Signs) != 2 || res.Signs[0].Name != "ctk-prod" || res.Signs[0].Status != "Approved" {
		t.Errorf("signs mismatch: %+v", res.Signs)
	}
	if res.Signs[1].Status != "Pending" {
		t.Errorf("expected Pending status, got %q", res.Signs[1].Status)
	}
	if len(res.Templates) != 1 || res.Templates[0].Name != "OTP" {
		t.Errorf("templates mismatch: %+v", res.Templates)
	}
}

func TestGetResourcePropagatesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Action":"DescribeUSMSSignatureResponse","RetCode":171,"Message":"signature mismatch"}`))
	}))
	defer server.Close()

	driver := newDriver(server.URL)
	_, err := driver.GetResource(context.Background())
	if err == nil {
		t.Fatal("expected error when DescribeUSMSSignature fails")
	}
	if !strings.Contains(err.Error(), "signature mismatch") {
		t.Errorf("expected error message in err, got %v", err)
	}
}

func TestGetResourceHandlesEmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		_, _ = w.Write([]byte(`{"Action":"` + r.Form.Get("Action") + `Response","RetCode":0,"TotalCount":0,"Data":[]}`))
	}))
	defer server.Close()

	driver := newDriver(server.URL)
	res, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource: %v", err)
	}
	if len(res.Signs) != 0 || len(res.Templates) != 0 {
		t.Errorf("expected empty result, got %+v", res)
	}
}
