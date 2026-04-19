package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	iamProjectAPIVersion = "2021-08-01"
	iamUserAPIVersion    = "2018-01-01"
)

type ListProjectsResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		Projects []IAMProject `json:"Projects"`
		Total    int32        `json:"Total"`
	} `json:"Result"`
}

type IAMProject struct {
	ProjectName string `json:"ProjectName"`
	AccountID   int64  `json:"AccountID"`
}

type ListUsersResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		UserMetadata []IAMUserMetadata `json:"UserMetadata"`
		Total        int32             `json:"Total"`
		Limit        int32             `json:"Limit"`
		Offset       int32             `json:"Offset"`
	} `json:"Result"`
}

type IAMUserMetadata struct {
	UserName   string `json:"UserName"`
	AccountID  int64  `json:"AccountId"`
	CreateDate string `json:"CreateDate"`
}

type GetLoginProfileResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		LoginProfile IAMLoginProfile `json:"LoginProfile"`
	} `json:"Result"`
}

type IAMLoginProfile struct {
	UserName      string `json:"UserName"`
	LastLoginDate string `json:"LastLoginDate"`
}

func (c *Client) ListProjects(ctx context.Context, region string) (ListProjectsResponse, error) {
	var out ListProjectsResponse
	err := c.DoOpenAPI(ctx, Request{
		Service:    "iam",
		Version:    iamProjectAPIVersion,
		Action:     "ListProjects",
		Method:     http.MethodGet,
		Region:     region,
		Path:       "/",
		Idempotent: true,
	}, &out)
	return out, err
}

func (c *Client) ListUsers(ctx context.Context, region string, limit, offset int32) (ListUsersResponse, error) {
	query := url.Values{}
	query.Set("Limit", strconv.FormatInt(int64(limit), 10))
	query.Set("Offset", strconv.FormatInt(int64(offset), 10))
	var out ListUsersResponse
	err := c.DoOpenAPI(ctx, Request{
		Service:    "iam",
		Version:    iamUserAPIVersion,
		Action:     "ListUsers",
		Method:     http.MethodGet,
		Region:     region,
		Path:       "/",
		Query:      query,
		Idempotent: true,
	}, &out)
	return out, err
}

func (c *Client) GetLoginProfile(ctx context.Context, region, userName string) (GetLoginProfileResponse, error) {
	query := url.Values{}
	query.Set("UserName", strings.TrimSpace(userName))
	var out GetLoginProfileResponse
	err := c.DoOpenAPI(ctx, Request{
		Service:    "iam",
		Version:    iamUserAPIVersion,
		Action:     "GetLoginProfile",
		Method:     http.MethodGet,
		Region:     region,
		Path:       "/",
		Query:      query,
		Idempotent: true,
	}, &out)
	return out, err
}
