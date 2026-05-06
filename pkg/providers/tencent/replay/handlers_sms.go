package replay

import (
	"fmt"
	"net/http"

	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
)

// handleSMS serves the cloudlist `sms` asset actions `DescribeSmsSignList` /
// `DescribeSmsTemplateList`. The driver passes explicit ID sets; the replay
// returns a single demo entry per call for any non-empty request.
func (t *transport) handleSMS(req *http.Request, action string) (*http.Response, error) {
	switch action {
	case "DescribeSmsSignList":
		resp := api.DescribeSmsSignListResponse{}
		resp.Response.RequestID = "req-replay-sms-describe-signs"
		statusCode := 0
		signID := uint64(1)
		signType := uint64(0)
		signName := "ctk-demo"
		createTime := int64(1745020800)
		intl := uint64(0)
		resp.Response.DescribeSignListStatusSet = []api.SmsSignDetail{{
			SignID:        &signID,
			SignName:      &signName,
			StatusCode:    &statusCode,
			CreateTime:    &createTime,
			International: &intl,
			SignType:      &signType,
		}}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	case "DescribeSmsTemplateList":
		resp := api.DescribeSmsTemplateListResponse{}
		resp.Response.RequestID = "req-replay-sms-describe-templates"
		statusCode := 0
		tplID := uint64(1)
		tplName := "ctk-demo-otp"
		content := "Your OTP is {1}"
		createTime := int64(1745020800)
		intl := uint64(0)
		resp.Response.DescribeTemplateStatusSet = []api.SmsTemplateDetail{{
			TemplateID:      &tplID,
			TemplateName:    &tplName,
			TemplateContent: &content,
			StatusCode:      &statusCode,
			CreateTime:      &createTime,
			International:   &intl,
		}}
		return demoreplay.JSONResponse(req, http.StatusOK, resp), nil
	}
	return openAPIErrorResponse(req, http.StatusNotFound, "InvalidAction.NotFound",
		fmt.Sprintf("Unsupported replay action: %s", action)), nil
}
