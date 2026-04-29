package api

import (
	"context"
	"encoding/json"
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
	UserName              string `json:"UserName"`
	LastLoginDate         string `json:"LastLoginDate"`
	LoginAllowed          bool   `json:"LoginAllowed"`
	PasswordResetRequired bool   `json:"PasswordResetRequired"`
}

type CreateUserResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		User IAMUserMetadata `json:"User"`
	} `json:"Result"`
}

type CreateLoginProfileResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		LoginProfile IAMLoginProfile `json:"LoginProfile"`
	} `json:"Result"`
}

type DeleteLoginProfileResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
}

type AttachUserPolicyResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
}

type DetachUserPolicyResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
}

type DeleteUserResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
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
	setTrimmedQueryValue(query, "UserName", userName)
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

func (c *Client) CreateUser(ctx context.Context, region, userName, displayName string) (CreateUserResponse, error) {
	query := url.Values{}
	setTrimmedQueryValue(query, "UserName", userName)
	if displayName = strings.TrimSpace(displayName); displayName != "" {
		query.Set("DisplayName", displayName)
	}
	var out CreateUserResponse
	err := c.DoOpenAPI(ctx, Request{
		Service: "iam",
		Version: iamUserAPIVersion,
		Action:  "CreateUser",
		Method:  http.MethodGet,
		Region:  region,
		Path:    "/",
		Query:   query,
	}, &out)
	return out, err
}

func (c *Client) CreateLoginProfile(ctx context.Context, region, userName, password string) (CreateLoginProfileResponse, error) {
	userName = strings.TrimSpace(userName)
	body, err := json.Marshal(struct {
		UserName              string `json:"UserName"`
		Password              string `json:"Password"`
		LoginAllowed          bool   `json:"LoginAllowed"`
		PasswordResetRequired bool   `json:"PasswordResetRequired"`
	}{
		UserName:              userName,
		Password:              password,
		LoginAllowed:          true,
		PasswordResetRequired: false,
	})
	if err != nil {
		return CreateLoginProfileResponse{}, err
	}
	var out CreateLoginProfileResponse
	err = c.DoOpenAPI(ctx, Request{
		Service: "iam",
		Version: iamUserAPIVersion,
		Action:  "CreateLoginProfile",
		Method:  http.MethodPost,
		Region:  region,
		Path:    "/",
		Body:    body,
	}, &out)
	return out, err
}

func (c *Client) DeleteLoginProfile(ctx context.Context, region, userName string) (DeleteLoginProfileResponse, error) {
	query := url.Values{}
	setTrimmedQueryValue(query, "UserName", userName)
	var out DeleteLoginProfileResponse
	err := c.DoOpenAPI(ctx, Request{
		Service: "iam",
		Version: iamUserAPIVersion,
		Action:  "DeleteLoginProfile",
		Method:  http.MethodGet,
		Region:  region,
		Path:    "/",
		Query:   query,
	}, &out)
	return out, err
}

func (c *Client) AttachUserPolicy(ctx context.Context, region, userName, policyName, policyType string) (AttachUserPolicyResponse, error) {
	query := url.Values{}
	setTrimmedQueryValue(query, "UserName", userName)
	setTrimmedQueryValue(query, "PolicyName", policyName)
	setTrimmedQueryValue(query, "PolicyType", policyType)
	var out AttachUserPolicyResponse
	err := c.DoOpenAPI(ctx, Request{
		Service: "iam",
		Version: iamUserAPIVersion,
		Action:  "AttachUserPolicy",
		Method:  http.MethodGet,
		Region:  region,
		Path:    "/",
		Query:   query,
	}, &out)
	return out, err
}

func (c *Client) DetachUserPolicy(ctx context.Context, region, userName, policyName, policyType string) (DetachUserPolicyResponse, error) {
	query := url.Values{}
	setTrimmedQueryValue(query, "UserName", userName)
	setTrimmedQueryValue(query, "PolicyName", policyName)
	setTrimmedQueryValue(query, "PolicyType", policyType)
	var out DetachUserPolicyResponse
	err := c.DoOpenAPI(ctx, Request{
		Service: "iam",
		Version: iamUserAPIVersion,
		Action:  "DetachUserPolicy",
		Method:  http.MethodGet,
		Region:  region,
		Path:    "/",
		Query:   query,
	}, &out)
	return out, err
}

func (c *Client) DeleteUser(ctx context.Context, region, userName string) (DeleteUserResponse, error) {
	query := url.Values{}
	setTrimmedQueryValue(query, "UserName", userName)
	var out DeleteUserResponse
	err := c.DoOpenAPI(ctx, Request{
		Service: "iam",
		Version: iamUserAPIVersion,
		Action:  "DeleteUser",
		Method:  http.MethodGet,
		Region:  region,
		Path:    "/",
		Query:   query,
	}, &out)
	return out, err
}

func setTrimmedQueryValue(query url.Values, key, value string) {
	query.Set(key, strings.TrimSpace(value))
}
