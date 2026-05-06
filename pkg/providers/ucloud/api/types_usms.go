package api

// UCloud USMS DescribeUSMSSignature / DescribeUSMSTemplate — pattern-inferred
// against UCloud's JSON-RPC convention (DescribeXxxSignature / Template
// neighboured the existing DescribeULogTopic family). Verify against
// upstream USMS docs before relying on this in production.

type USMSSignature struct {
	SigID      string `json:"SigId"`
	SigContent string `json:"SigContent"`
	Status     int    `json:"Status"`
	ErrMsg     string `json:"ErrMsg,omitempty"`
	UpdateTime int64  `json:"UpdateTime"`
	SigType    int    `json:"SigType"`
}

type DescribeUSMSSignatureResponse struct {
	BaseResponse
	TotalCount int             `json:"TotalCount"`
	Signatures []USMSSignature `json:"Data"`
}

type USMSTemplate struct {
	TemplateID   string `json:"TemplateId"`
	Template     string `json:"Template"`
	TemplateName string `json:"TemplateName"`
	TemplateType int    `json:"TemplateType"`
	Status       int    `json:"Status"`
	ErrMsg       string `json:"ErrMsg,omitempty"`
	UpdateTime   int64  `json:"UpdateTime"`
}

type DescribeUSMSTemplateResponse struct {
	BaseResponse
	TotalCount int            `json:"TotalCount"`
	Templates  []USMSTemplate `json:"Data"`
}
