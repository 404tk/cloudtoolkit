package api

// Huawei MSGSMS v2 — list SMS templates and signatures for the cloudlist `sms`
// asset. The response shape mirrors huaweicloud-sdk-go-v3 services/msgsms/v2:
//
//   GET /v2/{project_id}/msgsms/signatures
//   GET /v2/{project_id}/msgsms/templates

type MSGSMSSign struct {
	ID         string `json:"id"`
	SignID     string `json:"signature_id"`
	SignName   string `json:"signature_name"`
	Status     string `json:"status"`
	SignType   string `json:"signature_type"`
	CreateTime string `json:"create_time"`
	Reason     string `json:"review_desc"`
}

type ListSmsSignResponse struct {
	Results []MSGSMSSign `json:"results"`
	Total   int64        `json:"total"`
}

type MSGSMSTemplate struct {
	ID           string `json:"id"`
	TemplateID   string `json:"template_id"`
	TemplateName string `json:"template_name"`
	Content      string `json:"template_content"`
	Status       string `json:"status"`
	FlowStatus   string `json:"flow_status"`
	TemplateType string `json:"template_type"`
	CreateTime   string `json:"create_time"`
	Reason       string `json:"review_desc"`
}

type ListSmsTemplateResponse struct {
	Results []MSGSMSTemplate `json:"results"`
	Total   int64            `json:"total"`
}
