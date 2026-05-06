package replay

import (
	"net/http"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

// handleSMSAPI serves Huawei MSGSMS `/v1/sms/signs` and `/v1/sms/templates`
// used by the cloudlist `sms` asset.
func (t *transport) handleSMSAPI(req *http.Request, _ string) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return apiErrorResponse(req, http.StatusMethodNotAllowed, "SMS.0001",
			"unsupported smsapi method: "+req.Method), nil
	}
	switch req.URL.Path {
	case "/v1/sms/signs":
		resp := api.ListSmsSignResponse{
			Signs: []api.MSGSMSSign{
				{SignID: "sign-1", SignName: "ctk-demo", SignType: "ENTERPRISE", Status: "APPROVED", CreateTime: "2026-03-01 00:00:00"},
			},
		}
		resp.TotalCount = len(resp.Signs)
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "/v1/sms/templates":
		resp := api.ListSmsTemplateResponse{
			Templates: []api.MSGSMSTemplate{
				{TemplateID: "tpl-1", TemplateName: "ctk-demo-otp", Content: "Your OTP is ${1}", TemplateType: "VERIFICATION", Status: "APPROVED", CreateTime: "2026-03-01 00:00:00"},
			},
		}
		resp.TotalCount = len(resp.Templates)
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	}
	return apiErrorResponse(req, http.StatusNotFound, "SMS.0001",
		"unsupported smsapi path: "+req.URL.Path), nil
}
