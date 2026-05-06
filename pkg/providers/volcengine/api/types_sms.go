package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// Volcengine SMS — list templates and signs (audit assets). Pattern-inferred
// against Volcengine's documented OpenAPI v1 surface; verify against
// upstream SMS docs before relying on this in production.
const smsAPIVersion = "2020-01-01"

type SMSSign struct {
	SignID     string `json:"SignId"`
	Sign       string `json:"Sign"`
	SignType   string `json:"SignType"`
	Status     string `json:"Status"`
	Reason     string `json:"Reason"`
	CreateTime string `json:"CreateTime"`
}

type ListSmsSignResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		Total int       `json:"Total"`
		List  []SMSSign `json:"List"`
	} `json:"Result"`
}

type SMSTemplate struct {
	TemplateID   string `json:"TemplateId"`
	TemplateName string `json:"TemplateName"`
	TemplateType string `json:"TemplateType"`
	Content      string `json:"Content"`
	Status       string `json:"Status"`
	Reason       string `json:"Reason"`
	CreateTime   string `json:"CreateTime"`
}

type ListSmsTemplateResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		Total int           `json:"Total"`
		List  []SMSTemplate `json:"List"`
	} `json:"Result"`
}

func (c *Client) ListSmsSigns(ctx context.Context, region string, pageNumber, pageSize int) (ListSmsSignResponse, error) {
	query := url.Values{}
	if pageNumber > 0 {
		query.Set("PageNumber", strconv.Itoa(pageNumber))
	}
	if pageSize > 0 {
		query.Set("PageSize", strconv.Itoa(pageSize))
	}
	var out ListSmsSignResponse
	err := c.DoOpenAPI(ctx, Request{
		Service:    "volcsms",
		Version:    smsAPIVersion,
		Action:     "ListSign",
		Method:     http.MethodGet,
		Region:     region,
		Path:       "/",
		Query:      query,
		Idempotent: true,
	}, &out)
	return out, err
}

func (c *Client) ListSmsTemplates(ctx context.Context, region string, pageNumber, pageSize int) (ListSmsTemplateResponse, error) {
	query := url.Values{}
	if pageNumber > 0 {
		query.Set("PageNumber", strconv.Itoa(pageNumber))
	}
	if pageSize > 0 {
		query.Set("PageSize", strconv.Itoa(pageSize))
	}
	var out ListSmsTemplateResponse
	err := c.DoOpenAPI(ctx, Request{
		Service:    "volcsms",
		Version:    smsAPIVersion,
		Action:     "ListSmsTemplate",
		Method:     http.MethodGet,
		Region:     region,
		Path:       "/",
		Query:      query,
		Idempotent: true,
	}, &out)
	return out, err
}
