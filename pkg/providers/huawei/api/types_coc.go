package api

import (
	"context"
	"net/http"
	"strconv"
)

// Huawei COC (Cloud Operations Center) script execution API. UniAgent must be
// installed on the target ECS for executions to land.

type COCCreateScriptRequest struct {
	Name                string              `json:"name"`
	Properties          COCScriptProperties `json:"properties"`
	Description         string              `json:"description"`
	Type                string              `json:"type"`
	Content             string              `json:"content"`
	EnterpriseProjectID string              `json:"enterprise_project_id,omitempty"`
}

type COCScriptProperties struct {
	RiskLevel string `json:"risk_level"`
	Version   string `json:"version"`
}

type COCCreateScriptResponse struct {
	Data string `json:"data"`
}

type COCExecuteScriptRequest struct {
	ExecuteParam   COCScriptExecuteParam          `json:"execute_param"`
	ExecuteBatches []COCExecuteInstancesBatchInfo `json:"execute_batches"`
}

type COCScriptExecuteParam struct {
	Timeout     int32   `json:"timeout"`
	SuccessRate float64 `json:"success_rate"`
	ExecuteUser string  `json:"execute_user"`
}

type COCExecuteInstancesBatchInfo struct {
	BatchIndex       int32                        `json:"batch_index"`
	TargetInstances  []COCExecuteResourceInstance `json:"target_instances"`
	RotationStrategy string                       `json:"rotation_strategy"`
}

type COCExecuteResourceInstance struct {
	ResourceID string `json:"resource_id"`
	RegionID   string `json:"region_id"`
	Provider   string `json:"provider,omitempty"`
	Type       string `json:"type,omitempty"`
}

type COCExecuteScriptResponse struct {
	Data string `json:"data"`
}

type COCGetScriptJobInfoResponse struct {
	Data *COCJobScriptOrderInfo `json:"data,omitempty"`
}

type COCJobScriptOrderInfo struct {
	ExecuteUUID string                      `json:"execute_uuid,omitempty"`
	Status      string                      `json:"status,omitempty"`
	Properties  *COCJobScriptOrderInfoProps `json:"properties,omitempty"`
}

type COCJobScriptOrderInfoProps struct {
	CurrentExecuteBatchIndex int32 `json:"current_execute_batch_index,omitempty"`
}

type COCGetScriptJobBatchResponse struct {
	Data *COCJobScriptBatchDetail `json:"data,omitempty"`
}

type COCJobScriptBatchDetail struct {
	BatchIndex       int32                  `json:"batch_index,omitempty"`
	TotalInstances   int32                  `json:"total_instances,omitempty"`
	ExecuteInstances []COCExecutionInstance `json:"execute_instances,omitempty"`
}

type COCExecutionInstance struct {
	TargetInstance *COCResourceInstance `json:"target_instance,omitempty"`
	Status         string               `json:"status,omitempty"`
	Message        string               `json:"message,omitempty"`
}

type COCResourceInstance struct {
	ResourceID string `json:"resource_id,omitempty"`
	Provider   string `json:"provider,omitempty"`
	RegionID   string `json:"region_id,omitempty"`
	Type       string `json:"type,omitempty"`
}

type COCDeleteScriptResponse struct {
	Data string `json:"data,omitempty"`
}

func (c *Client) COCCreateScript(ctx context.Context, region, projectID string, body []byte) (COCCreateScriptResponse, error) {
	var resp COCCreateScriptResponse
	err := c.DoJSON(ctx, Request{
		Service: "coc",
		Region:  region,
		Method:  http.MethodPost,
		Path:    "/v1/job/scripts",
		Body:    body,
		Headers: cocHeaders(projectID),
	}, &resp)
	return resp, err
}

func (c *Client) COCExecuteScript(ctx context.Context, region, projectID, scriptUUID string, body []byte) (COCExecuteScriptResponse, error) {
	var resp COCExecuteScriptResponse
	err := c.DoJSON(ctx, Request{
		Service: "coc",
		Region:  region,
		Method:  http.MethodPost,
		Path:    "/v1/job/scripts/" + scriptUUID,
		Body:    body,
		Headers: cocHeaders(projectID),
	}, &resp)
	return resp, err
}

func (c *Client) COCGetScriptJobInfo(ctx context.Context, region, projectID, executeUUID string) (COCGetScriptJobInfoResponse, error) {
	var resp COCGetScriptJobInfoResponse
	err := c.DoJSON(ctx, Request{
		Service:    "coc",
		Region:     region,
		Method:     http.MethodGet,
		Path:       "/v1/job/script/orders/" + executeUUID,
		Headers:    cocHeaders(projectID),
		Idempotent: true,
	}, &resp)
	return resp, err
}

func (c *Client) COCGetScriptJobBatch(ctx context.Context, region, projectID, executeUUID string, batchIndex int32) (COCGetScriptJobBatchResponse, error) {
	var resp COCGetScriptJobBatchResponse
	err := c.DoJSON(ctx, Request{
		Service:    "coc",
		Region:     region,
		Method:     http.MethodGet,
		Path:       "/v1/job/script/orders/" + executeUUID + "/batches/" + strconv.FormatInt(int64(batchIndex), 10),
		Headers:    cocHeaders(projectID),
		Idempotent: true,
	}, &resp)
	return resp, err
}

func (c *Client) COCDeleteScript(ctx context.Context, region, projectID, scriptUUID string) (COCDeleteScriptResponse, error) {
	var resp COCDeleteScriptResponse
	err := c.DoJSON(ctx, Request{
		Service:    "coc",
		Region:     region,
		Method:     http.MethodDelete,
		Path:       "/v1/job/scripts/" + scriptUUID,
		Headers:    cocHeaders(projectID),
		Idempotent: true,
	}, &resp)
	return resp, err
}

func cocHeaders(projectID string) http.Header {
	headers := http.Header{}
	if projectID != "" {
		headers.Set("x-project-id", projectID)
	}
	return headers
}
