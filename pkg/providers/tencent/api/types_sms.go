package api

import "context"

// Tencent Cloud SMS — list templates and signs (audit assets).
const smsAPIVersion = "2021-01-11"

type DescribeSmsSignListRequest struct {
	SignIDSet []uint64 `json:"SignIdSet"`
	International *uint64 `json:"International,omitempty"`
}

type SmsSignDetail struct {
	SignID         *uint64 `json:"SignId"`
	SignName       *string `json:"SignName"`
	StatusCode     *int    `json:"StatusCode"`
	ReviewReply    *string `json:"ReviewReply"`
	CreateTime     *int64  `json:"CreateTime"`
	International  *uint64 `json:"International"`
	SignType       *uint64 `json:"SignType"`
}

type DescribeSmsSignListResponse struct {
	Response struct {
		DescribeSignListStatusSet []SmsSignDetail `json:"DescribeSignListStatusSet"`
		RequestID                 string          `json:"RequestId"`
	} `json:"Response"`
}

type DescribeSmsTemplateListRequest struct {
	TemplateIDSet []uint64 `json:"TemplateIdSet"`
	International *uint64  `json:"International,omitempty"`
}

type SmsTemplateDetail struct {
	TemplateID     *uint64 `json:"TemplateId"`
	TemplateName   *string `json:"TemplateName"`
	TemplateContent *string `json:"TemplateContent"`
	StatusCode     *int    `json:"StatusCode"`
	ReviewReply    *string `json:"ReviewReply"`
	CreateTime     *int64  `json:"CreateTime"`
	International  *uint64 `json:"International"`
}

type DescribeSmsTemplateListResponse struct {
	Response struct {
		DescribeTemplateStatusSet []SmsTemplateDetail `json:"DescribeTemplateStatusSet"`
		RequestID                 string              `json:"RequestId"`
	} `json:"Response"`
}

func (c *Client) DescribeSmsSignList(ctx context.Context, region string, signIDs []uint64) (DescribeSmsSignListResponse, error) {
	req := DescribeSmsSignListRequest{SignIDSet: signIDs}
	var resp DescribeSmsSignListResponse
	err := c.DoJSON(ctx, "sms", smsAPIVersion, "DescribeSmsSignList", region, req, &resp)
	return resp, err
}

func (c *Client) DescribeSmsTemplateList(ctx context.Context, region string, templateIDs []uint64) (DescribeSmsTemplateListResponse, error) {
	req := DescribeSmsTemplateListRequest{TemplateIDSet: templateIDs}
	var resp DescribeSmsTemplateListResponse
	err := c.DoJSON(ctx, "sms", smsAPIVersion, "DescribeSmsTemplateList", region, req, &resp)
	return resp, err
}
