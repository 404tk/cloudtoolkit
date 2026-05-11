package api

// JDCloud Logs service (`logs` / `jcs:logs`).

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// LogsetEnd is the SDK DescribeLogsets `LogsetEnd` shape.
type LogsetEnd struct {
	UID              string `json:"uID"`
	CreateTime       string `json:"createTime"`
	Description      string `json:"description"`
	HasTopic         bool   `json:"hasTopic"`
	LifeCycle        int64  `json:"lifeCycle"`
	Name             string `json:"name"`
	Region           string `json:"region"`
	ResourceGroupUID string `json:"resourceGroupUID"`
}

type DescribeLogsetsResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		Data          []LogsetEnd `json:"data"`
		NumberPages   int64       `json:"numberPages,omitempty"`
		NumberRecords int64       `json:"numberRecords,omitempty"`
		PageNumber    int64       `json:"pageNumber,omitempty"`
		PageSize      int64       `json:"pageSize,omitempty"`
	} `json:"result"`
}

// DescribeLogsets lists logsets in a JDCloud region.
func (c *Client) DescribeLogsets(ctx context.Context, region string, pageNumber, pageSize int) (DescribeLogsetsResponse, error) {
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
	var resp DescribeLogsetsResponse
	err := c.DoJSON(ctx, Request{
		Service:    "logs",
		Region:     region,
		Method:     http.MethodGet,
		Version:    "v1",
		Path:       "/regions/" + region + "/logsets",
		Query:      query,
		Idempotent: true,
	}, &resp)
	return resp, err
}
