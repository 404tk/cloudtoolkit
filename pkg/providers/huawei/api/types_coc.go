package api

import (
	"context"
	"net/http"
)

// Huawei COC (Cloud Operations Center) BatchExecuteCommand — pattern-inferred
// against the documented `coc.<region>.myhuaweicloud.com` v1 paths. The
// validation flow uses this as the closest equivalent to alibaba
// CloudAssistant / AWS SSM RunCommand. Verify against upstream COC docs
// before relying on this in production; UniAgent must be installed on the
// target ECS for executions to land.

type COCBatchExecuteCommandRequest struct {
	InstanceIDs   []string `json:"instance_ids"`
	WorkingDir    string   `json:"working_dir,omitempty"`
	Content       string   `json:"content"`
	ScriptType    string   `json:"script_type"`
	Username      string   `json:"username,omitempty"`
	ExecutionMode string   `json:"execution_mode,omitempty"`
	TimeoutSec    int64    `json:"execute_timeout,omitempty"`
}

type COCBatchExecuteCommandResponse struct {
	OrderID string `json:"order_id"`
	JobID   string `json:"job_id"`
	Status  string `json:"status"`
}

type COCJobInstanceResult struct {
	InstanceID string `json:"instance_id"`
	Status     string `json:"execute_status"`
	Output     string `json:"output"`
	Message    string `json:"message"`
}

type COCDescribeJobResponse struct {
	OrderID  string                 `json:"order_id"`
	JobID    string                 `json:"job_id"`
	Status   string                 `json:"status"`
	Results  []COCJobInstanceResult `json:"instances"`
}

// COCBatchExecuteCommand submits a script execution against `instanceIDs`.
// `script` is the raw shell content; the driver wraps it in /bin/bash.
func (c *Client) COCBatchExecuteCommand(ctx context.Context, region string, body []byte) (COCBatchExecuteCommandResponse, error) {
	var resp COCBatchExecuteCommandResponse
	err := c.DoJSON(ctx, Request{
		Service: "coc",
		Region:  region,
		Method:  http.MethodPost,
		Path:    "/v1/job/scripts/orders/batch-execute",
		Body:    body,
	}, &resp)
	return resp, err
}

// COCDescribeJob polls the status of a previously submitted execution.
func (c *Client) COCDescribeJob(ctx context.Context, region, orderID string) (COCDescribeJobResponse, error) {
	var resp COCDescribeJobResponse
	err := c.DoJSON(ctx, Request{
		Service:    "coc",
		Region:     region,
		Method:     http.MethodGet,
		Path:       "/v1/job/scripts/orders/" + orderID,
		Idempotent: true,
	}, &resp)
	return resp, err
}
