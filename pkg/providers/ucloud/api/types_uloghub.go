package api

// UCloud ULogHub (USLS) DescribeULogTopic — pattern-inferred against UCloud's
// JSON-RPC convention. The topic-list shape mirrors neighbouring DescribeUSMS
// / DescribeULogSet families; verify against the upstream SDK before relying
// on this in production.

type ULogTopic struct {
	TopicID    string `json:"TopicID"`
	TopicName  string `json:"TopicName"`
	LogSetID   string `json:"LogSetID"`
	LogSetName string `json:"LogSetName"`
	Region     string `json:"Region"`
	CreateTime int64  `json:"CreateTime"`
	UpdateTime int64  `json:"UpdateTime"`
}

type DescribeULogTopicResponse struct {
	BaseResponse
	TotalCount int         `json:"TotalCount"`
	Topics     []ULogTopic `json:"Topics"`
}
