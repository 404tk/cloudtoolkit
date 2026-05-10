package api

// UCloud Operation Log (ULog) GetUserOperationEvents.

type UCloudOperationEvent struct {
	Region          string                  `json:"Region"`
	API             string                  `json:"Api"`
	IsSuccess       bool                    `json:"IsSuccess"`
	OperateTime     int64                   `json:"OperateTime"`
	UserName        string                  `json:"UserName"`
	UserEmail       string                  `json:"UserEmail"`
	RelatedResource []UCloudRelatedResource `json:"RelatedResource"`
}

type UCloudRelatedResource struct {
	ResourceID   string `json:"ResourceId"`
	ResourceName string `json:"ResourceName"`
}

type GetUserOperationEventsResponse struct {
	BaseResponse
	NextToken string                 `json:"NextToken,omitempty"`
	Events    []UCloudOperationEvent `json:"Events"`
}
