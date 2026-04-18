package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type QuerySMSSignListResponse struct {
	RequestID   string        `json:"RequestId"`
	Code        string        `json:"Code"`
	Message     string        `json:"Message"`
	TotalCount  int64         `json:"TotalCount"`
	CurrentPage int           `json:"CurrentPage"`
	PageSize    int           `json:"PageSize"`
	SmsSignList []SMSSignInfo `json:"SmsSignList"`
}

type SMSSignInfo struct {
	SignName     string `json:"SignName"`
	AuditStatus  string `json:"AuditStatus"`
	BusinessType string `json:"BusinessType"`
}

func (c *Client) QuerySMSSignList(ctx context.Context, region string) (QuerySMSSignListResponse, error) {
	var resp QuerySMSSignListResponse
	err := c.Do(ctx, Request{
		Product:    "Dysmsapi",
		Version:    "2017-05-25",
		Action:     "QuerySmsSignList",
		Region:     region,
		Method:     http.MethodPost,
		Idempotent: true,
	}, &resp)
	if err != nil {
		return resp, err
	}
	return resp, decodeSMSResponse(resp.Code, resp.Message, resp.RequestID)
}

type QuerySMSTemplateListResponse struct {
	RequestID       string            `json:"RequestId"`
	Code            string            `json:"Code"`
	Message         string            `json:"Message"`
	TotalCount      int64             `json:"TotalCount"`
	CurrentPage     int               `json:"CurrentPage"`
	PageSize        int               `json:"PageSize"`
	SmsTemplateList []SMSTemplateInfo `json:"SmsTemplateList"`
}

type SMSTemplateInfo struct {
	TemplateName    string `json:"TemplateName"`
	AuditStatus     string `json:"AuditStatus"`
	TemplateContent string `json:"TemplateContent"`
}

func (c *Client) QuerySMSTemplateList(ctx context.Context, region string) (QuerySMSTemplateListResponse, error) {
	var resp QuerySMSTemplateListResponse
	err := c.Do(ctx, Request{
		Product:    "Dysmsapi",
		Version:    "2017-05-25",
		Action:     "QuerySmsTemplateList",
		Region:     region,
		Method:     http.MethodPost,
		Idempotent: true,
	}, &resp)
	if err != nil {
		return resp, err
	}
	return resp, decodeSMSResponse(resp.Code, resp.Message, resp.RequestID)
}

type QuerySMSSendStatisticsResponse struct {
	RequestID string                `json:"RequestId"`
	Code      string                `json:"Code"`
	Message   string                `json:"Message"`
	Data      SMSSendStatisticsData `json:"Data"`
}

type SMSSendStatisticsData struct {
	TotalSize int64 `json:"TotalSize"`
}

func (c *Client) QuerySMSSendStatistics(ctx context.Context, region, date string) (QuerySMSSendStatisticsResponse, error) {
	query := url.Values{}
	query.Set("IsGlobe", strconv.Itoa(1))
	query.Set("StartDate", date)
	query.Set("EndDate", date)
	query.Set("PageIndex", strconv.Itoa(1))
	query.Set("PageSize", strconv.Itoa(10))

	var resp QuerySMSSendStatisticsResponse
	err := c.Do(ctx, Request{
		Product:    "Dysmsapi",
		Version:    "2017-05-25",
		Action:     "QuerySendStatistics",
		Region:     region,
		Method:     http.MethodPost,
		Query:      query,
		Idempotent: true,
	}, &resp)
	if err != nil {
		return resp, err
	}
	return resp, decodeSMSResponse(resp.Code, resp.Message, resp.RequestID)
}

func decodeSMSResponse(code, message, requestID string) error {
	if strings.EqualFold(strings.TrimSpace(code), "OK") || strings.TrimSpace(code) == "" {
		return nil
	}
	return &APIError{
		Code:      code,
		Message:   message,
		RequestID: requestID,
	}
}
