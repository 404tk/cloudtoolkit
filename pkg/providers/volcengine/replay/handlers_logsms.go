package replay

import (
	"fmt"
	"net/http"

	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
)

// handleTLS serves the cloudlist `log` asset action `DescribeProjects` for the
// Volcengine TLS service.
func (t *transport) handleTLS(req *http.Request, action string) (*http.Response, error) {
	if action != "DescribeProjects" {
		return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction",
			fmt.Sprintf("unsupported tls action: %s", action)), nil
	}
	resp := api.DescribeTLSProjectsResponse{}
	resp.ResponseMetadata.RequestID = "req-tls-describe-projects"
	resp.Result.Projects = []api.TLSProject{
		{ProjectID: "tls-prod", ProjectName: "ctk-demo-app", Region: requestRegion(req), CreateTime: "2026-04-01 08:00:00", Description: "ctk demo application logs"},
		{ProjectID: "tls-audit", ProjectName: "ctk-demo-audit", Region: requestRegion(req), CreateTime: "2026-03-15 08:00:00", Description: "ctk demo audit pipeline"},
	}
	resp.Result.Total = len(resp.Result.Projects)
	return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
}

// handleSMS serves the cloudlist `sms` asset actions `ListSign` /
// `ListSmsTemplate` for the Volcengine SMS service (`volcsms`).
func (t *transport) handleSMS(req *http.Request, action string) (*http.Response, error) {
	switch action {
	case "ListSign":
		resp := api.ListSmsSignResponse{}
		resp.ResponseMetadata.RequestID = "req-sms-list-signs"
		resp.Result.List = []api.SMSSign{
			{SignID: "sg-1", Sign: "ctk-demo", SignType: "company", Status: "PASSED", CreateTime: "2026-03-01 00:00:00"},
		}
		resp.Result.Total = len(resp.Result.List)
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "ListSmsTemplate":
		resp := api.ListSmsTemplateResponse{}
		resp.ResponseMetadata.RequestID = "req-sms-list-templates"
		resp.Result.List = []api.SMSTemplate{
			{TemplateID: "tpl-1", TemplateName: "ctk-demo-otp", TemplateType: "verification", Content: "Your OTP is {1}", Status: "PASSED", CreateTime: "2026-03-01 00:00:00"},
		}
		resp.Result.Total = len(resp.Result.List)
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	}
	return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction",
		fmt.Sprintf("unsupported volcsms action: %s", action)), nil
}
