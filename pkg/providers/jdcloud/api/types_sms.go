package api

// JDCloud SMS — describeTemplates / describeSigns. Pattern-inferred against
// JDCloud's documented v1 surface; verify against upstream SMS docs before
// relying on this in production.

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

type SMSTemplate struct {
	TemplateID   string `json:"templateId"`
	TemplateName string `json:"templateName"`
	Content      string `json:"templateContent"`
	Status       string `json:"status"`
	Reason       string `json:"reason,omitempty"`
	CreateTime   string `json:"createTime"`
}

type DescribeTemplatesResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		Templates  []SMSTemplate `json:"templates"`
		TotalCount int           `json:"totalCount"`
	} `json:"result"`
}

type SMSSign struct {
	SignID     string `json:"signId"`
	SignName   string `json:"signName"`
	SignType   string `json:"signType"`
	Status     string `json:"status"`
	Reason     string `json:"reason,omitempty"`
	CreateTime string `json:"createTime"`
}

type DescribeSignsResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		Signs      []SMSSign `json:"signs"`
		TotalCount int       `json:"totalCount"`
	} `json:"result"`
}

func (c *Client) DescribeSmsTemplates(ctx context.Context, region string, pageNumber, pageSize int) (DescribeTemplatesResponse, error) {
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
	var resp DescribeTemplatesResponse
	err := c.DoJSON(ctx, Request{
		Service:    "sms",
		Region:     "",
		Method:     http.MethodGet,
		Version:    "v1",
		Path:       "/regions/" + region + "/templates",
		Query:      query,
		Idempotent: true,
	}, &resp)
	return resp, err
}

func (c *Client) DescribeSmsSigns(ctx context.Context, region string, pageNumber, pageSize int) (DescribeSignsResponse, error) {
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
	var resp DescribeSignsResponse
	err := c.DoJSON(ctx, Request{
		Service:    "sms",
		Region:     "",
		Method:     http.MethodGet,
		Version:    "v1",
		Path:       "/regions/" + region + "/signs",
		Query:      query,
		Idempotent: true,
	}, &resp)
	return resp, err
}
