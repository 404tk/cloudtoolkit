package api

// Huawei MSGSMS — list SMS templates and signs (audit assets). Pattern-
// inferred against the documented `smn.<region>.myhuaweicloud.com` /
// `smsapi.<region>.myhuaweicloud.com` v1 surface; verify against upstream
// MSGSMS docs before relying on this in production.

type MSGSMSSign struct {
	SignID     string `json:"sign_id"`
	SignName   string `json:"sign_name"`
	Status     string `json:"sign_status"`
	SignType   string `json:"sign_type"`
	CreateTime string `json:"create_time"`
	Reason     string `json:"reason"`
}

type ListSmsSignResponse struct {
	Signs      []MSGSMSSign `json:"signs"`
	TotalCount int          `json:"total_count"`
}

type MSGSMSTemplate struct {
	TemplateID   string `json:"template_id"`
	TemplateName string `json:"template_name"`
	Content      string `json:"content"`
	Status       string `json:"template_status"`
	TemplateType string `json:"template_type"`
	CreateTime   string `json:"create_time"`
	Reason       string `json:"reason"`
}

type ListSmsTemplateResponse struct {
	Templates  []MSGSMSTemplate `json:"templates"`
	TotalCount int              `json:"total_count"`
}
