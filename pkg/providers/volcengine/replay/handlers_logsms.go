package replay

import (
	"fmt"
	"net/http"

	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
)

// handleTLS serves the cloudlist `log` asset endpoint `/DescribeProjects` for
// the Volcengine TLS service.
func (t *transport) handleTLS(req *http.Request) (*http.Response, error) {
	if req.URL.Path != "/DescribeProjects" {
		return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction",
			fmt.Sprintf("unsupported tls path: %s", req.URL.Path)), nil
	}
	resp := api.DescribeTLSProjectsResponse{}
	resp.ResponseMetadata.RequestID = "req-tls-describe-projects"
	resp.Projects = []api.TLSProject{
		{ProjectID: "tls-prod", ProjectName: "ctk-demo-app", Region: requestRegion(req), CreateTime: "2026-04-01 08:00:00", Description: "ctk demo application logs"},
		{ProjectID: "tls-audit", ProjectName: "ctk-demo-audit", Region: requestRegion(req), CreateTime: "2026-03-15 08:00:00", Description: "ctk demo audit pipeline"},
	}
	resp.Total = len(resp.Projects)
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

// handleSMS serves the cloudlist `sms` asset actions for the Volcengine SMS
// service.
func (t *transport) handleSMS(req *http.Request, action string) (*http.Response, error) {
	switch action {
	case "GetSubAccountList":
		resp := api.ListSmsSubAccountsResponse{}
		resp.ResponseMetadata.RequestID = "req-sms-list-subaccounts"
		resp.Result.List = []api.SMSSubAccount{
			{SubAccountID: "sms-sub-1", SubAccountName: "ctk-validation", Status: 1, CreatedTime: 1772304000},
		}
		resp.Result.Total = len(resp.Result.List)
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "GetSignatureAndOrderList":
		resp := api.ListSmsSignResponse{}
		resp.ResponseMetadata.RequestID = "req-sms-list-signs"
		resp.Result.List = []api.SMSSign{
			{ID: "sg-1", Content: "ctk-demo", Source: "company", StatusCode: 3, CreatedAt: 1772304000},
		}
		resp.Result.Total = len(resp.Result.List)
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "GetSmsTemplateAndOrderList":
		resp := api.ListSmsTemplateResponse{}
		resp.ResponseMetadata.RequestID = "req-sms-list-templates"
		resp.Result.List = []api.SMSTemplate{
			{ID: "tpl-1", Name: "ctk-demo-otp", ChannelType: "verification", Text: "Your OTP is {1}", StatusCode: 3, CreatedAt: 1772304000},
		}
		resp.Result.Total = len(resp.Result.List)
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	}
	return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction",
		fmt.Sprintf("unsupported sms action: %s", action)), nil
}
