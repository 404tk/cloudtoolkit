package replay

import (
	"net/http"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
)

func (t *transport) handleDescribeULogTopic(req *http.Request) (*http.Response, error) {
	resp := api.DescribeULogTopicResponse{
		BaseResponse: newBase("DescribeULogTopicResponse"),
		TotalCount:   2,
		Topics: []api.ULogTopic{
			{TopicID: "topic-app-01", TopicName: "ctk-demo-app", LogSetID: "set-01", LogSetName: "ctk-demo", Region: "cn-bj2", CreateTime: 1745020800, UpdateTime: 1745107200},
			{TopicID: "topic-audit-02", TopicName: "ctk-demo-audit", LogSetID: "set-01", LogSetName: "ctk-demo", Region: "cn-bj2", CreateTime: 1744934400, UpdateTime: 1745020800},
		},
	}
	return successResponse(req, resp), nil
}

func (t *transport) handleDescribeUSMSSignature(req *http.Request) (*http.Response, error) {
	resp := api.DescribeUSMSSignatureResponse{
		BaseResponse: newBase("DescribeUSMSSignatureResponse"),
		TotalCount:   1,
		Signatures: []api.USMSSignature{
			{SigID: "sig-01", SigContent: "ctk-demo", Status: 0, SigType: 0, UpdateTime: 1745020800},
		},
	}
	return successResponse(req, resp), nil
}

func (t *transport) handleDescribeUSMSTemplate(req *http.Request) (*http.Response, error) {
	resp := api.DescribeUSMSTemplateResponse{
		BaseResponse: newBase("DescribeUSMSTemplateResponse"),
		TotalCount:   1,
		Templates: []api.USMSTemplate{
			{TemplateID: "tpl-01", TemplateName: "ctk-demo-otp", Template: "Your OTP is {1}", TemplateType: 0, Status: 0, UpdateTime: 1745020800},
		},
	}
	return successResponse(req, resp), nil
}
