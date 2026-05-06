package sms

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
)

func newDriver(baseURL string) *Driver {
	d := &Driver{
		Credential:  auth.New("ak", "sk", ""),
		Region:      "ap-guangzhou",
		SignIDs:     []uint64{1, 2},
		TemplateIDs: []uint64{100},
	}
	d.SetClientOptions(
		api.WithBaseURL(baseURL),
		api.WithClock(func() time.Time { return time.Unix(1776458501, 0).UTC() }),
		api.WithRetryPolicy(api.RetryPolicy{MaxAttempts: 1, Sleep: func(context.Context, time.Duration) error { return nil }}),
	)
	return d
}

func TestGetResourceMapsSignsAndTemplates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-TC-Action") {
		case "DescribeSmsSignList":
			_, _ = w.Write([]byte(`{"Response":{"DescribeSignListStatusSet":[
  {"SignId":1,"SignName":"ctk-prod","SignType":2,"StatusCode":0,"ReviewReply":"OK"},
  {"SignId":2,"SignName":"ctk-promo","SignType":1,"StatusCode":1,"ReviewReply":"missing license"}
],"RequestId":"r1"}}`))
		case "DescribeSmsTemplateList":
			_, _ = w.Write([]byte(`{"Response":{"DescribeTemplateStatusSet":[
  {"TemplateId":100,"TemplateName":"OTP","TemplateContent":"Code is {1}","StatusCode":0}
],"RequestId":"r2"}}`))
		default:
			t.Fatalf("unexpected action: %s", r.Header.Get("X-TC-Action"))
		}
	}))
	defer server.Close()

	driver := newDriver(server.URL)
	res, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource: %v", err)
	}
	if len(res.Signs) != 2 {
		t.Fatalf("expected 2 signs, got %d", len(res.Signs))
	}
	if res.Signs[0].Name != "ctk-prod" || res.Signs[0].Status != "Approved" || res.Signs[0].Type != "Website" {
		t.Errorf("sign mismatch: %+v", res.Signs[0])
	}
	if res.Signs[1].Status != "Rejected" {
		t.Errorf("expected Rejected, got %q", res.Signs[1].Status)
	}
	if len(res.Templates) != 1 || res.Templates[0].Name != "OTP" || res.Templates[0].Status != "Approved" {
		t.Errorf("template mismatch: %+v", res.Templates)
	}
}

func TestGetResourceWithoutIDsIsNoOp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("should not call API when SignIDs+TemplateIDs are empty")
	}))
	defer server.Close()

	driver := newDriver(server.URL)
	driver.SignIDs = nil
	driver.TemplateIDs = nil
	res, err := driver.GetResource(context.Background())
	if err != nil {
		t.Fatalf("GetResource: %v", err)
	}
	if len(res.Signs) != 0 || len(res.Templates) != 0 {
		t.Errorf("expected empty result, got %+v", res)
	}
}

func TestGetResourcePropagatesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Response":{"Error":{"Code":"AuthFailure.SignatureFailure","Message":"signature mismatch"},"RequestId":"r-err"}}`))
	}))
	defer server.Close()

	driver := newDriver(server.URL)
	_, err := driver.GetResource(context.Background())
	if err == nil {
		t.Fatal("expected error when DescribeSmsSignList returns Error")
	}
}
