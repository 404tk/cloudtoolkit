package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

type ListRAMUsersResponse struct {
	RequestID   string      `json:"RequestId"`
	IsTruncated bool        `json:"IsTruncated"`
	Marker      string      `json:"Marker"`
	Users       RAMUserList `json:"Users"`
}

type RAMUserList struct {
	User []RAMUser `json:"User"`
}

type RAMUser struct {
	MobilePhone   string `json:"MobilePhone"`
	Comments      string `json:"Comments"`
	CreateDate    string `json:"CreateDate"`
	AttachDate    string `json:"AttachDate"`
	Email         string `json:"Email"`
	UserID        string `json:"UserId"`
	UpdateDate    string `json:"UpdateDate"`
	UserName      string `json:"UserName"`
	JoinDate      string `json:"JoinDate"`
	LastLoginDate string `json:"LastLoginDate"`
	DisplayName   string `json:"DisplayName"`
}

func (c *Client) ListRAMUsers(ctx context.Context, region, marker string, maxItems int) (ListRAMUsersResponse, error) {
	query := url.Values{}
	if marker != "" {
		query.Set("Marker", marker)
	}
	if maxItems > 0 {
		query.Set("MaxItems", strconv.Itoa(maxItems))
	}

	var resp ListRAMUsersResponse
	err := c.Do(ctx, Request{
		Product:    "Ram",
		Version:    "2015-05-01",
		Action:     "ListUsers",
		Region:     region,
		Method:     http.MethodPost,
		Query:      query,
		Idempotent: true,
	}, &resp)
	return resp, err
}

type GetRAMLoginProfileResponse struct {
	RequestID    string          `json:"RequestId"`
	LoginProfile RAMLoginProfile `json:"LoginProfile"`
}

type RAMLoginProfile struct {
	MFABindRequired       bool   `json:"MFABindRequired"`
	CreateDate            string `json:"CreateDate"`
	UserName              string `json:"UserName"`
	PasswordResetRequired bool   `json:"PasswordResetRequired"`
}

func (c *Client) GetRAMLoginProfile(ctx context.Context, region, userName string) (GetRAMLoginProfileResponse, error) {
	query := url.Values{}
	query.Set("UserName", userName)

	var resp GetRAMLoginProfileResponse
	err := c.Do(ctx, Request{
		Product:    "Ram",
		Version:    "2015-05-01",
		Action:     "GetLoginProfile",
		Region:     region,
		Method:     http.MethodPost,
		Query:      query,
		Idempotent: true,
	}, &resp)
	return resp, err
}

type GetRAMUserResponse struct {
	RequestID string  `json:"RequestId"`
	User      RAMUser `json:"User"`
}

func (c *Client) GetRAMUser(ctx context.Context, region, userName string) (GetRAMUserResponse, error) {
	query := url.Values{}
	query.Set("UserName", userName)

	var resp GetRAMUserResponse
	err := c.Do(ctx, Request{
		Product:    "Ram",
		Version:    "2015-05-01",
		Action:     "GetUser",
		Region:     region,
		Method:     http.MethodPost,
		Query:      query,
		Idempotent: true,
	}, &resp)
	return resp, err
}

type ListRAMPoliciesForUserResponse struct {
	RequestID string        `json:"RequestId"`
	Policies  RAMPolicyList `json:"Policies"`
}

type RAMPolicyList struct {
	Policy []RAMPolicy `json:"Policy"`
}

type RAMPolicy struct {
	PolicyDocument  string `json:"PolicyDocument"`
	CreateDate      string `json:"CreateDate"`
	AttachDate      string `json:"AttachDate"`
	PolicyType      string `json:"PolicyType"`
	UpdateDate      string `json:"UpdateDate"`
	AttachmentCount int    `json:"AttachmentCount"`
	DefaultVersion  string `json:"DefaultVersion"`
	PolicyName      string `json:"PolicyName"`
	Description     string `json:"Description"`
}

func (c *Client) ListRAMPoliciesForUser(ctx context.Context, region, userName string) (ListRAMPoliciesForUserResponse, error) {
	query := url.Values{}
	query.Set("UserName", userName)

	var resp ListRAMPoliciesForUserResponse
	err := c.Do(ctx, Request{
		Product:    "Ram",
		Version:    "2015-05-01",
		Action:     "ListPoliciesForUser",
		Region:     region,
		Method:     http.MethodPost,
		Query:      query,
		Idempotent: true,
	}, &resp)
	return resp, err
}

type GetRAMPolicyResponse struct {
	RequestID            string                  `json:"RequestId"`
	Policy               RAMPolicy               `json:"Policy"`
	DefaultPolicyVersion RAMDefaultPolicyVersion `json:"DefaultPolicyVersion"`
}

type RAMDefaultPolicyVersion struct {
	IsDefaultVersion bool   `json:"IsDefaultVersion"`
	PolicyDocument   string `json:"PolicyDocument"`
	VersionID        string `json:"VersionId"`
	CreateDate       string `json:"CreateDate"`
}

func (c *Client) GetRAMPolicy(ctx context.Context, region, policyName, policyType string) (GetRAMPolicyResponse, error) {
	query := url.Values{}
	query.Set("PolicyName", policyName)
	query.Set("PolicyType", policyType)

	var resp GetRAMPolicyResponse
	err := c.Do(ctx, Request{
		Product:    "Ram",
		Version:    "2015-05-01",
		Action:     "GetPolicy",
		Region:     region,
		Method:     http.MethodPost,
		Query:      query,
		Idempotent: true,
	}, &resp)
	return resp, err
}

type GetRAMAccountAliasResponse struct {
	RequestID    string `json:"RequestId"`
	AccountAlias string `json:"AccountAlias"`
}

func (c *Client) GetRAMAccountAlias(ctx context.Context, region string) (GetRAMAccountAliasResponse, error) {
	var resp GetRAMAccountAliasResponse
	err := c.Do(ctx, Request{
		Product:    "Ram",
		Version:    "2015-05-01",
		Action:     "GetAccountAlias",
		Region:     region,
		Method:     http.MethodPost,
		Idempotent: true,
	}, &resp)
	return resp, err
}

type CreateRAMUserResponse struct {
	RequestID string  `json:"RequestId"`
	User      RAMUser `json:"User"`
}

func (c *Client) CreateRAMUser(ctx context.Context, region, userName string) (CreateRAMUserResponse, error) {
	query := url.Values{}
	query.Set("UserName", userName)

	var resp CreateRAMUserResponse
	err := c.Do(ctx, Request{
		Product: "Ram",
		Version: "2015-05-01",
		Action:  "CreateUser",
		Region:  region,
		Method:  http.MethodPost,
		Query:   query,
	}, &resp)
	return resp, err
}

type CreateRAMLoginProfileResponse struct {
	RequestID    string          `json:"RequestId"`
	LoginProfile RAMLoginProfile `json:"LoginProfile"`
}

func (c *Client) CreateRAMLoginProfile(ctx context.Context, region, userName, password string) (CreateRAMLoginProfileResponse, error) {
	query := url.Values{}
	query.Set("UserName", userName)
	query.Set("Password", password)

	var resp CreateRAMLoginProfileResponse
	err := c.Do(ctx, Request{
		Product: "Ram",
		Version: "2015-05-01",
		Action:  "CreateLoginProfile",
		Region:  region,
		Method:  http.MethodPost,
		Query:   query,
	}, &resp)
	return resp, err
}

type AttachRAMPolicyToUserResponse struct {
	RequestID string `json:"RequestId"`
}

func (c *Client) AttachRAMPolicyToUser(ctx context.Context, region, userName, policyName, policyType string) (AttachRAMPolicyToUserResponse, error) {
	query := url.Values{}
	query.Set("UserName", userName)
	query.Set("PolicyName", policyName)
	query.Set("PolicyType", policyType)

	var resp AttachRAMPolicyToUserResponse
	err := c.Do(ctx, Request{
		Product: "Ram",
		Version: "2015-05-01",
		Action:  "AttachPolicyToUser",
		Region:  region,
		Method:  http.MethodPost,
		Query:   query,
	}, &resp)
	return resp, err
}

type DetachRAMPolicyFromUserResponse struct {
	RequestID string `json:"RequestId"`
}

func (c *Client) DetachRAMPolicyFromUser(ctx context.Context, region, userName, policyName, policyType string) (DetachRAMPolicyFromUserResponse, error) {
	query := url.Values{}
	query.Set("UserName", userName)
	query.Set("PolicyName", policyName)
	query.Set("PolicyType", policyType)

	var resp DetachRAMPolicyFromUserResponse
	err := c.Do(ctx, Request{
		Product: "Ram",
		Version: "2015-05-01",
		Action:  "DetachPolicyFromUser",
		Region:  region,
		Method:  http.MethodPost,
		Query:   query,
	}, &resp)
	return resp, err
}

type DeleteRAMUserResponse struct {
	RequestID string `json:"RequestId"`
}

func (c *Client) DeleteRAMUser(ctx context.Context, region, userName string) (DeleteRAMUserResponse, error) {
	query := url.Values{}
	query.Set("UserName", userName)

	var resp DeleteRAMUserResponse
	err := c.Do(ctx, Request{
		Product: "Ram",
		Version: "2015-05-01",
		Action:  "DeleteUser",
		Region:  region,
		Method:  http.MethodPost,
		Query:   query,
	}, &resp)
	return resp, err
}

type CreateRAMRoleResponse struct {
	RequestID string  `json:"RequestId"`
	Role      RAMRole `json:"Role"`
}

type RAMRole struct {
	CreateDate               string `json:"CreateDate"`
	RoleID                   string `json:"RoleId"`
	AttachDate               string `json:"AttachDate"`
	Arn                      string `json:"Arn"`
	UpdateDate               string `json:"UpdateDate"`
	MaxSessionDuration       int64  `json:"MaxSessionDuration"`
	Description              string `json:"Description"`
	AssumeRolePolicyDocument string `json:"AssumeRolePolicyDocument"`
	RoleName                 string `json:"RoleName"`
}

func (c *Client) CreateRAMRole(ctx context.Context, region, roleName, assumeRolePolicyDocument string) (CreateRAMRoleResponse, error) {
	query := url.Values{}
	query.Set("RoleName", roleName)
	query.Set("AssumeRolePolicyDocument", assumeRolePolicyDocument)

	var resp CreateRAMRoleResponse
	err := c.Do(ctx, Request{
		Product: "Ram",
		Version: "2015-05-01",
		Action:  "CreateRole",
		Region:  region,
		Method:  http.MethodPost,
		Query:   query,
	}, &resp)
	return resp, err
}

type AttachRAMPolicyToRoleResponse struct {
	RequestID string `json:"RequestId"`
}

func (c *Client) AttachRAMPolicyToRole(ctx context.Context, region, roleName, policyName, policyType string) (AttachRAMPolicyToRoleResponse, error) {
	query := url.Values{}
	query.Set("RoleName", roleName)
	query.Set("PolicyName", policyName)
	query.Set("PolicyType", policyType)

	var resp AttachRAMPolicyToRoleResponse
	err := c.Do(ctx, Request{
		Product: "Ram",
		Version: "2015-05-01",
		Action:  "AttachPolicyToRole",
		Region:  region,
		Method:  http.MethodPost,
		Query:   query,
	}, &resp)
	return resp, err
}

type DetachRAMPolicyFromRoleResponse struct {
	RequestID string `json:"RequestId"`
}

func (c *Client) DetachRAMPolicyFromRole(ctx context.Context, region, roleName, policyName, policyType string) (DetachRAMPolicyFromRoleResponse, error) {
	query := url.Values{}
	query.Set("RoleName", roleName)
	query.Set("PolicyName", policyName)
	query.Set("PolicyType", policyType)

	var resp DetachRAMPolicyFromRoleResponse
	err := c.Do(ctx, Request{
		Product: "Ram",
		Version: "2015-05-01",
		Action:  "DetachPolicyFromRole",
		Region:  region,
		Method:  http.MethodPost,
		Query:   query,
	}, &resp)
	return resp, err
}

type DeleteRAMRoleResponse struct {
	RequestID string `json:"RequestId"`
}

func (c *Client) DeleteRAMRole(ctx context.Context, region, roleName string) (DeleteRAMRoleResponse, error) {
	query := url.Values{}
	query.Set("RoleName", roleName)

	var resp DeleteRAMRoleResponse
	err := c.Do(ctx, Request{
		Product: "Ram",
		Version: "2015-05-01",
		Action:  "DeleteRole",
		Region:  region,
		Method:  http.MethodPost,
		Query:   query,
	}, &resp)
	return resp, err
}
