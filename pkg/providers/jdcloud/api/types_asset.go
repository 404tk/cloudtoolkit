package api

type DescribeAccountAmountResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		TotalAmount          string `json:"totalAmount"`
		AvailableAmount      string `json:"availableAmount"`
		FrozenAmount         string `json:"frozenAmount"`
		EnableWithdrawAmount string `json:"enableWithdrawAmount"`
		WithdrawingAmount    string `json:"withdrawingAmount"`
	} `json:"result"`
}
