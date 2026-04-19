package api

type DescribeSubUsersResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		SubUsers []SubUser `json:"subUsers"`
		Total    int       `json:"total"`
	} `json:"result"`
}

type DescribeSubUserResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		SubUser SubUser `json:"subUser"`
	} `json:"result"`
}

type SubUser struct {
	Pin        string `json:"pin"`
	Name       string `json:"name"`
	Account    string `json:"account"`
	CreateTime string `json:"createTime"`
}
