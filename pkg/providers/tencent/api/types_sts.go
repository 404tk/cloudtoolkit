package api

type GetCallerIdentityRequest struct{}

type GetCallerIdentityResponse struct {
	Response struct {
		Arn       string `json:"Arn"`
		Type      string `json:"Type"`
		UserID    string `json:"UserId"`
		RequestID string `json:"RequestId"`
	} `json:"Response"`
}
