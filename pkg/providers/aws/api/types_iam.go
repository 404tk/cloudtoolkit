package api

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const iamAPIVersion = "2010-05-08"

type IAMUser struct {
	UserName         string
	UserID           string
	Arn              string
	CreateDate       *time.Time
	PasswordLastUsed *time.Time
}

type ListUsersOutput struct {
	Users       []IAMUser
	Marker      string
	IsTruncated bool
	RequestID   string
}

type AttachedUserPolicy struct {
	PolicyName string
	PolicyArn  string
}

type ListAttachedUserPoliciesOutput struct {
	Policies    []AttachedUserPolicy
	Marker      string
	IsTruncated bool
	RequestID   string
}

type GetLoginProfileOutput struct {
	CreateDate            *time.Time
	PasswordResetRequired bool
	RequestID             string
}

type CreateUserOutput struct {
	Arn       string
	RequestID string
}

type iamResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

type iamUserWire struct {
	UserName         string `xml:"UserName"`
	UserID           string `xml:"UserId"`
	Arn              string `xml:"Arn"`
	CreateDate       string `xml:"CreateDate"`
	PasswordLastUsed string `xml:"PasswordLastUsed"`
}

type listUsersResponse struct {
	XMLName         xml.Name            `xml:"ListUsersResponse"`
	ListUsersResult listUsersResult     `xml:"ListUsersResult"`
	Metadata        iamResponseMetadata `xml:"ResponseMetadata"`
}

type listUsersResult struct {
	Users       []iamUserWire `xml:"Users>member"`
	IsTruncated bool          `xml:"IsTruncated"`
	Marker      string        `xml:"Marker"`
}

type getLoginProfileResponse struct {
	XMLName               xml.Name              `xml:"GetLoginProfileResponse"`
	GetLoginProfileResult getLoginProfileResult `xml:"GetLoginProfileResult"`
	Metadata              iamResponseMetadata   `xml:"ResponseMetadata"`
}

type getLoginProfileResult struct {
	LoginProfile loginProfileWire `xml:"LoginProfile"`
}

type loginProfileWire struct {
	CreateDate            string `xml:"CreateDate"`
	PasswordResetRequired bool   `xml:"PasswordResetRequired"`
}

type listAttachedUserPoliciesResponse struct {
	XMLName                        xml.Name                       `xml:"ListAttachedUserPoliciesResponse"`
	ListAttachedUserPoliciesResult listAttachedUserPoliciesResult `xml:"ListAttachedUserPoliciesResult"`
	Metadata                       iamResponseMetadata            `xml:"ResponseMetadata"`
}

type listAttachedUserPoliciesResult struct {
	Policies    []attachedUserPolicyWire `xml:"AttachedPolicies>member"`
	IsTruncated bool                     `xml:"IsTruncated"`
	Marker      string                   `xml:"Marker"`
}

type attachedUserPolicyWire struct {
	PolicyName string `xml:"PolicyName"`
	PolicyArn  string `xml:"PolicyArn"`
}

type createUserResponse struct {
	XMLName          xml.Name            `xml:"CreateUserResponse"`
	CreateUserResult createUserResult    `xml:"CreateUserResult"`
	Metadata         iamResponseMetadata `xml:"ResponseMetadata"`
}

type createUserResult struct {
	User iamUserWire `xml:"User"`
}

func (c *Client) ListUsers(ctx context.Context, region, marker string) (ListUsersOutput, error) {
	query := url.Values{}
	if marker = strings.TrimSpace(marker); marker != "" {
		query.Set("Marker", marker)
	}
	var wire listUsersResponse
	err := c.DoXML(ctx, Request{
		Service:    "iam",
		Region:     region,
		Action:     "ListUsers",
		Version:    iamAPIVersion,
		Method:     http.MethodPost,
		Path:       "/",
		Query:      query,
		Idempotent: true,
	}, &wire)
	if err != nil {
		return ListUsersOutput{}, err
	}
	out := ListUsersOutput{
		Users:       make([]IAMUser, 0, len(wire.ListUsersResult.Users)),
		Marker:      strings.TrimSpace(wire.ListUsersResult.Marker),
		IsTruncated: wire.ListUsersResult.IsTruncated,
		RequestID:   strings.TrimSpace(wire.Metadata.RequestID),
	}
	for _, user := range wire.ListUsersResult.Users {
		out.Users = append(out.Users, IAMUser{
			UserName:         strings.TrimSpace(user.UserName),
			UserID:           strings.TrimSpace(user.UserID),
			Arn:              strings.TrimSpace(user.Arn),
			CreateDate:       parseAWSTime(user.CreateDate),
			PasswordLastUsed: parseAWSTime(user.PasswordLastUsed),
		})
	}
	return out, nil
}

func (c *Client) GetLoginProfile(ctx context.Context, region, userName string) (GetLoginProfileOutput, error) {
	query := url.Values{}
	query.Set("UserName", strings.TrimSpace(userName))
	var wire getLoginProfileResponse
	err := c.DoXML(ctx, Request{
		Service:    "iam",
		Region:     region,
		Action:     "GetLoginProfile",
		Version:    iamAPIVersion,
		Method:     http.MethodPost,
		Path:       "/",
		Query:      query,
		Idempotent: true,
	}, &wire)
	if err != nil {
		return GetLoginProfileOutput{}, err
	}
	return GetLoginProfileOutput{
		CreateDate:            parseAWSTime(wire.GetLoginProfileResult.LoginProfile.CreateDate),
		PasswordResetRequired: wire.GetLoginProfileResult.LoginProfile.PasswordResetRequired,
		RequestID:             strings.TrimSpace(wire.Metadata.RequestID),
	}, nil
}

func (c *Client) ListAttachedUserPolicies(ctx context.Context, region, userName, marker string) (ListAttachedUserPoliciesOutput, error) {
	query := url.Values{}
	query.Set("UserName", strings.TrimSpace(userName))
	if marker = strings.TrimSpace(marker); marker != "" {
		query.Set("Marker", marker)
	}
	var wire listAttachedUserPoliciesResponse
	err := c.DoXML(ctx, Request{
		Service:    "iam",
		Region:     region,
		Action:     "ListAttachedUserPolicies",
		Version:    iamAPIVersion,
		Method:     http.MethodPost,
		Path:       "/",
		Query:      query,
		Idempotent: true,
	}, &wire)
	if err != nil {
		return ListAttachedUserPoliciesOutput{}, err
	}
	out := ListAttachedUserPoliciesOutput{
		Policies:    make([]AttachedUserPolicy, 0, len(wire.ListAttachedUserPoliciesResult.Policies)),
		Marker:      strings.TrimSpace(wire.ListAttachedUserPoliciesResult.Marker),
		IsTruncated: wire.ListAttachedUserPoliciesResult.IsTruncated,
		RequestID:   strings.TrimSpace(wire.Metadata.RequestID),
	}
	for _, policy := range wire.ListAttachedUserPoliciesResult.Policies {
		out.Policies = append(out.Policies, AttachedUserPolicy{
			PolicyName: strings.TrimSpace(policy.PolicyName),
			PolicyArn:  strings.TrimSpace(policy.PolicyArn),
		})
	}
	return out, nil
}

func (c *Client) CreateUser(ctx context.Context, region, userName string) (CreateUserOutput, error) {
	query := url.Values{}
	query.Set("UserName", strings.TrimSpace(userName))
	var wire createUserResponse
	err := c.DoXML(ctx, Request{
		Service: "iam",
		Region:  region,
		Action:  "CreateUser",
		Version: iamAPIVersion,
		Method:  http.MethodPost,
		Path:    "/",
		Query:   query,
	}, &wire)
	if err != nil {
		return CreateUserOutput{}, err
	}
	return CreateUserOutput{
		Arn:       strings.TrimSpace(wire.CreateUserResult.User.Arn),
		RequestID: strings.TrimSpace(wire.Metadata.RequestID),
	}, nil
}

func (c *Client) CreateLoginProfile(ctx context.Context, region, userName, password string) error {
	query := url.Values{}
	query.Set("UserName", strings.TrimSpace(userName))
	query.Set("Password", password)
	return c.DoXML(ctx, Request{
		Service: "iam",
		Region:  region,
		Action:  "CreateLoginProfile",
		Version: iamAPIVersion,
		Method:  http.MethodPost,
		Path:    "/",
		Query:   query,
	}, nil)
}

func (c *Client) AttachUserPolicy(ctx context.Context, region, userName, policyArn string) error {
	query := url.Values{}
	query.Set("UserName", strings.TrimSpace(userName))
	query.Set("PolicyArn", strings.TrimSpace(policyArn))
	return c.DoXML(ctx, Request{
		Service: "iam",
		Region:  region,
		Action:  "AttachUserPolicy",
		Version: iamAPIVersion,
		Method:  http.MethodPost,
		Path:    "/",
		Query:   query,
	}, nil)
}

func (c *Client) DetachUserPolicy(ctx context.Context, region, userName, policyArn string) error {
	query := url.Values{}
	query.Set("UserName", strings.TrimSpace(userName))
	query.Set("PolicyArn", strings.TrimSpace(policyArn))
	return c.DoXML(ctx, Request{
		Service: "iam",
		Region:  region,
		Action:  "DetachUserPolicy",
		Version: iamAPIVersion,
		Method:  http.MethodPost,
		Path:    "/",
		Query:   query,
	}, nil)
}

func (c *Client) DeleteLoginProfile(ctx context.Context, region, userName string) error {
	query := url.Values{}
	query.Set("UserName", strings.TrimSpace(userName))
	return c.DoXML(ctx, Request{
		Service: "iam",
		Region:  region,
		Action:  "DeleteLoginProfile",
		Version: iamAPIVersion,
		Method:  http.MethodPost,
		Path:    "/",
		Query:   query,
	}, nil)
}

func (c *Client) DeleteUser(ctx context.Context, region, userName string) error {
	query := url.Values{}
	query.Set("UserName", strings.TrimSpace(userName))
	return c.DoXML(ctx, Request{
		Service: "iam",
		Region:  region,
		Action:  "DeleteUser",
		Version: iamAPIVersion,
		Method:  http.MethodPost,
		Path:    "/",
		Query:   query,
	}, nil)
}

func normalizeIAMRegion(region string) string {
	region = normalizeRegion(region)
	if strings.HasPrefix(region, "cn-") {
		return "cn-north-1"
	}
	return "us-east-1"
}

func parseAWSTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	return &parsed
}
