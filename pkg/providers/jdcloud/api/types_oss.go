package api

type ListBucketsResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		Buckets []Bucket `json:"buckets"`
	} `json:"result"`
}

type Bucket struct {
	Name string `json:"name"`
}
