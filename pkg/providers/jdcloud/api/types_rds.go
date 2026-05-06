package api

// JDCloud RDS account lifecycle. Endpoint paths follow the standard
// `/v1/regions/<region>/instances/<id>/<resource>` pattern used elsewhere in
// JDCloud's REST API; verify against the upstream SDK before relying on
// this in production.

import (
	"context"
	"net/http"
)

type RDSAccount struct {
	AccountName       string `json:"accountName"`
	AccountStatus     string `json:"accountStatus,omitempty"`
	AccountType       string `json:"accountType,omitempty"`
	HostList          []string `json:"hostList,omitempty"`
	DatabasePrivileges []map[string]string `json:"databasePrivileges,omitempty"`
}

type DescribeRDSAccountsResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		Accounts []RDSAccount `json:"accounts"`
	} `json:"result"`
}

type CreateRDSAccountRequest struct {
	AccountName     string `json:"accountName"`
	AccountPassword string `json:"accountPassword"`
}

type CreateRDSAccountResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct{}      `json:"result"`
}

type DeleteRDSAccountResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct{}      `json:"result"`
}

func (c *Client) DescribeRDSAccounts(ctx context.Context, region, instanceID string) (DescribeRDSAccountsResponse, error) {
	if region == "" || region == "all" {
		region = "cn-north-1"
	}
	var resp DescribeRDSAccountsResponse
	err := c.DoJSON(ctx, Request{
		Service: "rds",
		Region:  region,
		Method:  http.MethodGet,
		Version: "v1",
		Path:    "/regions/" + region + "/instances/" + instanceID + "/accounts",
	}, &resp)
	return resp, err
}

func (c *Client) CreateRDSAccount(ctx context.Context, region, instanceID string, body []byte) (CreateRDSAccountResponse, error) {
	if region == "" || region == "all" {
		region = "cn-north-1"
	}
	var resp CreateRDSAccountResponse
	err := c.DoJSON(ctx, Request{
		Service: "rds",
		Region:  region,
		Method:  http.MethodPost,
		Version: "v1",
		Path:    "/regions/" + region + "/instances/" + instanceID + "/accounts",
		Body:    body,
	}, &resp)
	return resp, err
}

func (c *Client) DeleteRDSAccount(ctx context.Context, region, instanceID, accountName string) (DeleteRDSAccountResponse, error) {
	if region == "" || region == "all" {
		region = "cn-north-1"
	}
	var resp DeleteRDSAccountResponse
	err := c.DoJSON(ctx, Request{
		Service: "rds",
		Region:  region,
		Method:  http.MethodDelete,
		Version: "v1",
		Path:    "/regions/" + region + "/instances/" + instanceID + "/accounts/" + accountName,
	}, &resp)
	return resp, err
}
