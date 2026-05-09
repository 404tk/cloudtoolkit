package replay

import (
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

// handleSMSAPI serves Huawei MSGSMS v2 signature and template endpoints used by
// the cloudlist `sms` asset.
func (t *transport) handleSMSAPI(req *http.Request, _ string) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return apiErrorResponse(req, http.StatusMethodNotAllowed, "SMS.0001",
			"unsupported msgsms method: "+req.Method), nil
	}
	switch {
	case strings.HasSuffix(req.URL.Path, "/msgsms/signatures"):
		resp := api.ListSmsSignResponse{
			Results: []api.MSGSMSSign{
				{SignID: "sign-1", SignName: "ctk-demo", SignType: "ENTERPRISE", Status: "APPROVED", CreateTime: "2026-03-01 00:00:00"},
			},
		}
		resp.Total = int64(len(resp.Results))
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case strings.HasSuffix(req.URL.Path, "/msgsms/templates"):
		resp := api.ListSmsTemplateResponse{
			Results: []api.MSGSMSTemplate{
				{TemplateID: "tpl-1", TemplateName: "ctk-demo-otp", Content: "Your OTP is ${1}", TemplateType: "VERIFICATION", Status: "APPROVED", CreateTime: "2026-03-01 00:00:00"},
			},
		}
		resp.Total = int64(len(resp.Results))
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	}
	return apiErrorResponse(req, http.StatusNotFound, "SMS.0001",
		"unsupported msgsms path: "+req.URL.Path), nil
}
