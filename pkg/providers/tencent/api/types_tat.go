package api

import "context"

const tatVersion = "2020-10-28"

type RunTATCommandRequest struct {
	Content     *string  `json:"Content,omitempty"`
	InstanceIDs []string `json:"InstanceIds,omitempty"`
	CommandType *string  `json:"CommandType,omitempty"`
}

type RunTATCommandResponse struct {
	Response struct {
		CommandID    *string `json:"CommandId"`
		InvocationID *string `json:"InvocationId"`
		RequestID    string  `json:"RequestId"`
	} `json:"Response"`
}

func (c *Client) RunTATCommand(ctx context.Context, region, commandType, content string, instanceIDs []string) (RunTATCommandResponse, error) {
	var resp RunTATCommandResponse
	err := c.DoJSON(
		ctx,
		"tat",
		tatVersion,
		"RunCommand",
		normalizeRegion(region),
		RunTATCommandRequest{
			Content:     stringPtr(content),
			InstanceIDs: instanceIDs,
			CommandType: stringPtr(commandType),
		},
		&resp,
	)
	return resp, err
}

type DescribeTATInvocationsRequest struct {
	InvocationIDs []string `json:"InvocationIds,omitempty"`
}

type DescribeTATInvocationsResponse struct {
	Response struct {
		InvocationSet []TATInvocation `json:"InvocationSet"`
		RequestID     string          `json:"RequestId"`
	} `json:"Response"`
}

type TATInvocation struct {
	InvocationID               *string                  `json:"InvocationId"`
	InvocationTaskBasicInfoSet []TATInvocationTaskBasic `json:"InvocationTaskBasicInfoSet"`
}

type TATInvocationTaskBasic struct {
	InvocationTaskID *string `json:"InvocationTaskId"`
	TaskStatus       *string `json:"TaskStatus"`
	InstanceID       *string `json:"InstanceId"`
}

func (c *Client) DescribeTATInvocations(ctx context.Context, region string, invocationIDs []string) (DescribeTATInvocationsResponse, error) {
	var resp DescribeTATInvocationsResponse
	err := c.DoJSON(
		ctx,
		"tat",
		tatVersion,
		"DescribeInvocations",
		normalizeRegion(region),
		DescribeTATInvocationsRequest{InvocationIDs: invocationIDs},
		&resp,
	)
	return resp, err
}

type DescribeTATInvocationTasksRequest struct {
	InvocationTaskIDs []string `json:"InvocationTaskIds,omitempty"`
	HideOutput        *bool    `json:"HideOutput,omitempty"`
}

type DescribeTATInvocationTasksResponse struct {
	Response struct {
		InvocationTaskSet []TATInvocationTask `json:"InvocationTaskSet"`
		RequestID         string              `json:"RequestId"`
	} `json:"Response"`
}

type TATInvocationTask struct {
	InvocationID     *string        `json:"InvocationId"`
	InvocationTaskID *string        `json:"InvocationTaskId"`
	TaskStatus       *string        `json:"TaskStatus"`
	InstanceID       *string        `json:"InstanceId"`
	TaskResult       *TATTaskResult `json:"TaskResult"`
	ErrorInfo        *string        `json:"ErrorInfo"`
}

type TATTaskResult struct {
	ExitCode *int64  `json:"ExitCode"`
	Output   *string `json:"Output"`
}

func (c *Client) DescribeTATInvocationTasks(ctx context.Context, region string, invocationTaskIDs []string, hideOutput bool) (DescribeTATInvocationTasksResponse, error) {
	var resp DescribeTATInvocationTasksResponse
	err := c.DoJSON(
		ctx,
		"tat",
		tatVersion,
		"DescribeInvocationTasks",
		normalizeRegion(region),
		DescribeTATInvocationTasksRequest{
			InvocationTaskIDs: invocationTaskIDs,
			HideOutput:        boolPtr(hideOutput),
		},
		&resp,
	)
	return resp, err
}

func boolPtr(v bool) *bool {
	return &v
}
