package api

type CreateCommandRequest struct {
	RegionID           string `json:"regionId"`
	CommandName        string `json:"commandName"`
	CommandType        string `json:"commandType,omitempty"`
	CommandContent     string `json:"commandContent"`
	Timeout            int    `json:"timeout,omitempty"`
	Username           string `json:"username,omitempty"`
	Workdir            string `json:"workdir,omitempty"`
	CommandDescription string `json:"commandDescription,omitempty"`
	EnableParameter    *bool  `json:"enableParameter,omitempty"`
}

type CreateCommandResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		CommandID string `json:"commandId"`
	} `json:"result"`
}

type InvokeCommandRequest struct {
	RegionID  string   `json:"regionId"`
	CommandID string   `json:"commandId"`
	Instances []string `json:"instances,omitempty"`
	Timeout   int      `json:"timeout,omitempty"`
	Username  string   `json:"username,omitempty"`
	Workdir   string   `json:"workdir,omitempty"`
}

type InvokeCommandResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		InvokeID string `json:"invokeId"`
	} `json:"result"`
}

type DescribeInvocationsRequest struct {
	RegionID   string   `json:"regionId"`
	PageNumber int      `json:"pageNumber,omitempty"`
	PageSize   int      `json:"pageSize,omitempty"`
	InvokeIDs  []string `json:"invokeIds,omitempty"`
}

type DescribeInvocationsResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		TotalCount  int          `json:"totalCount"`
		Invocations []Invocation `json:"invocations"`
	} `json:"result"`
}

// Invocation mirrors the assistant's Invocation model. Only the fields used by
// the console validation flow are kept; the rest of the schema is omitted.
type Invocation struct {
	Status              string               `json:"status"`
	CommandID           string               `json:"commandId"`
	InvokeID            string               `json:"invokeId"`
	CommandType         string               `json:"commandType"`
	InvocationInstances []InvocationInstance `json:"invocationInstances"`
	ErrorInfo           string               `json:"errorInfo"`
	CreateTime          string               `json:"createTime"`
}

type InvocationInstance struct {
	InstanceID string `json:"instanceId"`
	Status     string `json:"status"`
	ExitCode   string `json:"exitCode"`
	ErrorInfo  string `json:"errorInfo"`
	StartTime  string `json:"startTime"`
	EndTime    string `json:"endTime"`
	Output     string `json:"output"`
}

type DeleteCommandsRequest struct {
	RegionID   string   `json:"regionId"`
	CommandIDs []string `json:"commandIds"`
}

type DeleteCommandsResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		CommandID string `json:"commandId"`
	} `json:"result"`
}
