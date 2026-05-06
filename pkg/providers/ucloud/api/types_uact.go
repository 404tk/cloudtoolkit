package api

// UCloud Action Trail / 操作审计 — pattern-inferred against UCloud's regular
// JSON-RPC convention. The action name follows UCloud's `Describe...` family;
// verify against the upstream SDK before relying on this in production.

type UActEvent struct {
	EventID         string `json:"EventId"`
	EventName       string `json:"EventName"`
	EventTime       string `json:"EventTime"`
	EventSource     string `json:"EventSource"`
	UserName        string `json:"UserName"`
	SourceIPAddress string `json:"SourceIPAddress"`
	Region          string `json:"Region"`
	Status          string `json:"Status"`
	AccessKey       string `json:"AccessKeyId"`
	ResourceName    string `json:"ResourceName"`
	ResourceType    string `json:"ResourceType"`
}

type DescribeActionLogListResponse struct {
	BaseResponse
	TotalCount int         `json:"TotalCount"`
	Events     []UActEvent `json:"Events"`
	NextToken  string      `json:"NextToken,omitempty"`
}
