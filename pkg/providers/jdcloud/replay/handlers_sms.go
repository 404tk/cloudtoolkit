package replay

import (
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
)

// handleSMS serves the cloudlist `sms` asset endpoints used by the JDCloud
// SMS driver: `/v1/regions/<region>/signs` and `/.../templates`.
func (t *transport) handleSMS(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return apiErrorResponse(req, http.StatusMethodNotAllowed, "MethodNotAllowed",
			"sms replay only supports GET"), nil
	}
	switch {
	case strings.HasSuffix(req.URL.Path, "/signs"):
		resp := api.DescribeSignsResponse{RequestID: "req-replay-sms-describe-signs"}
		resp.Result.Signs = []api.SMSSign{
			{SignID: "sign-1", SignName: "ctk-demo", SignType: "Enterprise", Status: "Approved", CreateTime: "2026-03-01T00:00:00Z"},
		}
		resp.Result.TotalCount = len(resp.Result.Signs)
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case strings.HasSuffix(req.URL.Path, "/templates"):
		resp := api.DescribeTemplatesResponse{RequestID: "req-replay-sms-describe-templates"}
		resp.Result.Templates = []api.SMSTemplate{
			{TemplateID: "tpl-1", TemplateName: "ctk-demo-otp", Content: "Your OTP is ${code}", Status: "Approved", CreateTime: "2026-03-01T00:00:00Z"},
		}
		resp.Result.TotalCount = len(resp.Result.Templates)
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	}
	return apiErrorResponse(req, http.StatusNotFound, "InvalidPath",
		"unsupported sms path: "+req.URL.Path), nil
}
