package api

type DescribeInstancesResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		Instances  []Instance `json:"instances"`
		TotalCount int        `json:"totalCount"`
	} `json:"result"`
}

type Instance struct {
	InstanceID       string `json:"instanceId"`
	Hostname         string `json:"hostname"`
	Status           string `json:"status"`
	OSType           string `json:"osType"`
	PrivateIPAddress string `json:"privateIpAddress"`
	ElasticIPAddress string `json:"elasticIpAddress"`
}
