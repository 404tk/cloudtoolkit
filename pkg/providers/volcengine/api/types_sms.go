package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

const (
	smsSignAPIVersion     = "2025-01-01"
	smsTemplateAPIVersion = "2021-01-11"
	smsSigningRegion      = "cn-north-1"
	smsSigningService     = "volcSMS"
)

type SMSSign struct {
	SignID     string `json:"SignId"`
	ID         string `json:"id"`
	Sign       string `json:"Sign"`
	Content    string `json:"content"`
	SignType   string `json:"SignType"`
	Source     string `json:"source"`
	Status     string `json:"Status"`
	StatusCode int64  `json:"status"`
	Reason     string `json:"Reason"`
	ReasonText string `json:"reason"`
	CreateTime string `json:"CreateTime"`
	CreatedAt  int64  `json:"createdTime"`
}

type ListSmsSignResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		Total       int       `json:"total"`
		LegacyTotal int       `json:"Total"`
		List        []SMSSign `json:"list"`
		LegacyList  []SMSSign `json:"List"`
	} `json:"Result"`
}

type SMSTemplate struct {
	TemplateID      string `json:"TemplateId"`
	TemplateIDLower string `json:"templateId"`
	ID              string `json:"id"`
	TemplateName    string `json:"TemplateName"`
	Name            string `json:"name"`
	TemplateType    string `json:"TemplateType"`
	Type            string `json:"type"`
	ChannelType     string `json:"channelType"`
	ChannelTypeName string `json:"channelTypeName"`
	Content         string `json:"Content"`
	Text            string `json:"content"`
	Status          string `json:"Status"`
	StatusCode      int64  `json:"status"`
	Reason          string `json:"Reason"`
	ReasonText      string `json:"reason"`
	CreateTime      string `json:"CreateTime"`
	CreatedAt       int64  `json:"createdTime"`
}

type ListSmsTemplateResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		Total       int           `json:"total"`
		LegacyTotal int           `json:"Total"`
		List        []SMSTemplate `json:"list"`
		LegacyList  []SMSTemplate `json:"List"`
	} `json:"Result"`
}

type SMSSubAccount struct {
	SubAccountID   string `json:"subAccountId"`
	SubAccount     string `json:"subAccount"`
	SubAccountName string `json:"subAccountName"`
	Status         int    `json:"status"`
	Desc           string `json:"desc"`
	CreatedTime    int64  `json:"createdTime"`
}

type ListSmsSubAccountsResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		Total int             `json:"total"`
		List  []SMSSubAccount `json:"list"`
	} `json:"Result"`
}

func (r ListSmsSignResponse) Signs() []SMSSign {
	if len(r.Result.List) > 0 || r.Result.Total > 0 {
		return r.Result.List
	}
	return r.Result.LegacyList
}

func (r ListSmsTemplateResponse) Templates() []SMSTemplate {
	if len(r.Result.List) > 0 || r.Result.Total > 0 {
		return r.Result.List
	}
	return r.Result.LegacyList
}

func (c *Client) ListSmsSubAccounts(ctx context.Context, pageNumber, pageSize int) (ListSmsSubAccountsResponse, error) {
	query := url.Values{}
	if pageNumber > 0 {
		query.Set("pageIndex", strconv.Itoa(pageNumber))
	}
	if pageSize > 0 {
		query.Set("pageSize", strconv.Itoa(pageSize))
	}
	var out ListSmsSubAccountsResponse
	err := c.DoOpenAPI(ctx, Request{
		Service:     "sms",
		SignService: smsSigningService,
		Version:     smsTemplateAPIVersion,
		Action:      "GetSubAccountList",
		Method:      http.MethodGet,
		Region:      smsSigningRegion,
		Path:        "/",
		Query:       query,
		Idempotent:  true,
	}, &out)
	return out, err
}

func (c *Client) ListSmsSigns(ctx context.Context, subAccount string, pageNumber, pageSize int) (ListSmsSignResponse, error) {
	query := url.Values{}
	if pageNumber > 0 {
		query.Set("pageIndex", strconv.Itoa(pageNumber))
	}
	if pageSize > 0 {
		query.Set("pageSize", strconv.Itoa(pageSize))
	}
	setTrimmedQueryValue(query, "subAccount", subAccount)
	setTrimmedQueryValue(query, "subAccounts", subAccount)
	var out ListSmsSignResponse
	err := c.DoOpenAPI(ctx, Request{
		Service:     "sms",
		SignService: smsSigningService,
		Version:     smsSignAPIVersion,
		Action:      "GetSignatureAndOrderList",
		Method:      http.MethodGet,
		Region:      smsSigningRegion,
		Path:        "/",
		Query:       query,
		Idempotent:  true,
	}, &out)
	return out, err
}

func (c *Client) ListSmsTemplates(ctx context.Context, subAccount string, pageNumber, pageSize int) (ListSmsTemplateResponse, error) {
	query := url.Values{}
	if pageNumber > 0 {
		query.Set("pageIndex", strconv.Itoa(pageNumber))
	}
	if pageSize > 0 {
		query.Set("pageSize", strconv.Itoa(pageSize))
	}
	setTrimmedQueryValue(query, "subAccount", subAccount)
	query.Set("area", "all")
	var out ListSmsTemplateResponse
	err := c.DoOpenAPI(ctx, Request{
		Service:     "sms",
		SignService: smsSigningService,
		Version:     smsTemplateAPIVersion,
		Action:      "GetSmsTemplateAndOrderList",
		Method:      http.MethodGet,
		Region:      smsSigningRegion,
		Path:        "/",
		Query:       query,
		Idempotent:  true,
	}, &out)
	return out, err
}
