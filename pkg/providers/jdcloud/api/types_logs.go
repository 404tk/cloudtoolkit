package api

// JDCloud Logs service (`logs` / `jcs:logs`) — describeLogTopics.
//
// Pattern-inferred against the JDCloud REST convention used by neighbouring
// services in this codebase (asset, actiontrail, oss): list paths follow
// `/v1/regions/<region>/<resource>:<action>` with snake-cased query params.
// Verify against the upstream JDCloud OpenAPI / SDK before relying on this
// in production deployments; the cloudlist `log` asset path keeps the demo
// surface working today.

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

type LogTopic struct {
	LogTopicID   string `json:"logTopicId"`
	LogTopicName string `json:"logTopicName"`
	Description  string `json:"description"`
	CreateTime   string `json:"createTime"`
	UpdateTime   string `json:"updateTime"`
	LogSetID     string `json:"logSetId"`
	LogSetName   string `json:"logSetName"`
}

type DescribeLogTopicsResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		Topics     []LogTopic `json:"logTopics"`
		TotalCount int        `json:"totalCount,omitempty"`
		PageNumber int        `json:"pageNumber,omitempty"`
		PageSize   int        `json:"pageSize,omitempty"`
	} `json:"result"`
}

// DescribeLogTopics lists log topics in a JDCloud region.
func (c *Client) DescribeLogTopics(ctx context.Context, region string, pageNumber, pageSize int) (DescribeLogTopicsResponse, error) {
	if region == "" || region == "all" {
		region = "cn-north-1"
	}
	query := url.Values{}
	if pageNumber > 0 {
		query.Set("pageNumber", strconv.Itoa(pageNumber))
	}
	if pageSize > 0 {
		query.Set("pageSize", strconv.Itoa(pageSize))
	}
	var resp DescribeLogTopicsResponse
	err := c.DoJSON(ctx, Request{
		Service: "logs",
		Region:  "",
		Method:  http.MethodGet,
		Version: "v1",
		Path:    "/regions/" + region + "/logTopics:describe",
		Query:   query,
	}, &resp)
	return resp, err
}
