package api

import (
	"context"
	"net/http"
	"net/url"
)

type DescribeRDSRegionsResponse struct {
	RequestID string        `json:"RequestId"`
	Regions   RDSRegionList `json:"Regions"`
}

type RDSRegionList struct {
	RDSRegion []RDSRegion `json:"RDSRegion"`
}

type RDSRegion struct {
	RegionID string `json:"RegionId"`
}

func (c *Client) DescribeRDSRegions(ctx context.Context, region string) (DescribeRDSRegionsResponse, error) {
	query := url.Values{}
	query.Set("AcceptLanguage", "en-US")

	var resp DescribeRDSRegionsResponse
	err := c.Do(ctx, Request{
		Product:    "Rds",
		Version:    "2014-08-15",
		Action:     "DescribeRegions",
		Region:     region,
		Method:     http.MethodPost,
		Query:      query,
		Idempotent: true,
	}, &resp)
	return resp, err
}

type DescribeRDSInstancesResponse struct {
	RequestID        string          `json:"RequestId"`
	PageNumber       int             `json:"PageNumber"`
	PageRecordCount  int             `json:"PageRecordCount"`
	TotalRecordCount int             `json:"TotalRecordCount"`
	Items            RDSInstanceList `json:"Items"`
}

type RDSInstanceList struct {
	DBInstance []RDSInstance `json:"DBInstance"`
}

type RDSInstance struct {
	DBInstanceID        string `json:"DBInstanceId"`
	Engine              string `json:"Engine"`
	EngineVersion       string `json:"EngineVersion"`
	RegionID            string `json:"RegionId"`
	ConnectionString    string `json:"ConnectionString"`
	InstanceNetworkType string `json:"InstanceNetworkType"`
}

func (c *Client) DescribeRDSInstances(ctx context.Context, region string, pageNumber, pageSize int) (DescribeRDSInstancesResponse, error) {
	var resp DescribeRDSInstancesResponse
	err := c.Do(ctx, Request{
		Product:    "Rds",
		Version:    "2014-08-15",
		Action:     "DescribeDBInstances",
		Region:     region,
		Method:     http.MethodPost,
		Query:      pagingQuery(pageNumber, pageSize),
		Idempotent: true,
	}, &resp)
	return resp, err
}

type DescribeRDSDatabasesResponse struct {
	RequestID string          `json:"RequestId"`
	Databases RDSDatabaseList `json:"Databases"`
}

type RDSDatabaseList struct {
	Database []RDSDatabase `json:"Database"`
}

type RDSDatabase struct {
	DBName string `json:"DBName"`
}

func (c *Client) DescribeRDSDatabases(ctx context.Context, region, instanceID string) (DescribeRDSDatabasesResponse, error) {
	query := pagingQuery(1, 30)
	query.Set("DBInstanceId", instanceID)
	query.Set("DBStatus", "Running")

	var resp DescribeRDSDatabasesResponse
	err := c.Do(ctx, Request{
		Product:    "Rds",
		Version:    "2014-08-15",
		Action:     "DescribeDatabases",
		Region:     region,
		Method:     http.MethodPost,
		Query:      query,
		Idempotent: true,
	}, &resp)
	return resp, err
}

type CreateRDSAccountResponse struct {
	RequestID string `json:"RequestId"`
}

func (c *Client) CreateRDSAccount(ctx context.Context, region, instanceID, accountName, password string) (CreateRDSAccountResponse, error) {
	query := url.Values{}
	query.Set("DBInstanceId", instanceID)
	query.Set("AccountName", accountName)
	query.Set("AccountPassword", password)
	query.Set("AccountType", "Normal")

	var resp CreateRDSAccountResponse
	err := c.Do(ctx, Request{
		Product: "Rds",
		Version: "2014-08-15",
		Action:  "CreateAccount",
		Region:  region,
		Method:  http.MethodPost,
		Query:   query,
	}, &resp)
	return resp, err
}

type GrantRDSAccountPrivilegeResponse struct {
	RequestID string `json:"RequestId"`
}

func (c *Client) GrantRDSAccountPrivilege(ctx context.Context, region, instanceID, accountName, dbName, privilege string) (GrantRDSAccountPrivilegeResponse, error) {
	query := url.Values{}
	query.Set("DBInstanceId", instanceID)
	query.Set("AccountName", accountName)
	query.Set("DBName", dbName)
	query.Set("AccountPrivilege", privilege)

	var resp GrantRDSAccountPrivilegeResponse
	err := c.Do(ctx, Request{
		Product: "Rds",
		Version: "2014-08-15",
		Action:  "GrantAccountPrivilege",
		Region:  region,
		Method:  http.MethodPost,
		Query:   query,
	}, &resp)
	return resp, err
}

type DeleteRDSAccountResponse struct {
	RequestID string `json:"RequestId"`
}

func (c *Client) DeleteRDSAccount(ctx context.Context, region, instanceID, accountName string) (DeleteRDSAccountResponse, error) {
	query := url.Values{}
	query.Set("DBInstanceId", instanceID)
	query.Set("AccountName", accountName)

	var resp DeleteRDSAccountResponse
	err := c.Do(ctx, Request{
		Product: "Rds",
		Version: "2014-08-15",
		Action:  "DeleteAccount",
		Region:  region,
		Method:  http.MethodPost,
		Query:   query,
	}, &resp)
	return resp, err
}
