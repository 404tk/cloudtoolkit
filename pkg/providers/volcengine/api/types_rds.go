package api

import (
	"context"
	"encoding/json"
	"net/http"
)

const (
	rdsAPIVersion        = "2022-01-01"
	ServiceRDSMySQL      = "rds_mysql"
	ServiceRDSPostgreSQL = "rds_postgresql"
	ServiceRDSMSSQL      = "rds_mssql"
)

type DescribeRDSRegionsResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		Regions []RDSRegion `json:"Regions"`
	} `json:"Result"`
}

type RDSRegion struct {
	RegionID   string `json:"RegionId"`
	RegionName string `json:"RegionName"`
}

type DescribeRDSMySQLInstancesResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		Instances []RDSMySQLInstance `json:"Instances"`
		Total     int32              `json:"Total"`
	} `json:"Result"`
}

type RDSMySQLInstance struct {
	AddressObject   []RDSAddressObject `json:"AddressObject"`
	DBEngineVersion string             `json:"DBEngineVersion"`
	InstanceID      string             `json:"InstanceId"`
	InstanceName    string             `json:"InstanceName"`
	InstanceStatus  string             `json:"InstanceStatus"`
	RegionID        string             `json:"RegionId"`
}

type DescribeRDSPostgreSQLInstancesResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		Instances []RDSPostgreSQLInstance `json:"Instances"`
		Total     int32                   `json:"Total"`
	} `json:"Result"`
}

type RDSPostgreSQLInstance struct {
	AddressObject   []RDSAddressObject `json:"AddressObject"`
	DBEngineVersion string             `json:"DBEngineVersion"`
	InstanceID      string             `json:"InstanceId"`
	InstanceName    string             `json:"InstanceName"`
	InstanceStatus  string             `json:"InstanceStatus"`
	RegionID        string             `json:"RegionId"`
}

type RDSAddressObject struct {
	Domain      string `json:"Domain"`
	IPAddress   string `json:"IPAddress"`
	NetworkType string `json:"NetworkType"`
	Port        string `json:"Port"`
}

type DescribeRDSSQLServerInstancesResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		InstancesInfo []RDSSQLServerInstance `json:"InstancesInfo"`
		Total         int32                  `json:"Total"`
	} `json:"Result"`
}

type RDSSQLServerInstance struct {
	DBEngineVersion string             `json:"DBEngineVersion"`
	InstanceID      string             `json:"InstanceId"`
	InstanceName    string             `json:"InstanceName"`
	InstanceStatus  string             `json:"InstanceStatus"`
	NodeDetailInfo  []RDSSQLServerNode `json:"NodeDetailInfo"`
	Port            string             `json:"Port"`
	RegionID        string             `json:"RegionId"`
}

type RDSSQLServerNode struct {
	NodeIP   string `json:"NodeIP"`
	NodeType string `json:"NodeType"`
}

type describeRDSInstancesInput struct {
	PageNumber int32 `json:"PageNumber"`
	PageSize   int32 `json:"PageSize"`
}

func (c *Client) DescribeRDSRegions(ctx context.Context, service, region string) (DescribeRDSRegionsResponse, error) {
	var out DescribeRDSRegionsResponse
	err := c.doRDSAction(ctx, service, "DescribeRegions", region, struct{}{}, &out)
	return out, err
}

func (c *Client) DescribeRDSMySQLInstances(ctx context.Context, region string, pageNumber, pageSize int32) (DescribeRDSMySQLInstancesResponse, error) {
	var out DescribeRDSMySQLInstancesResponse
	err := c.doRDSAction(ctx, ServiceRDSMySQL, "DescribeDBInstances", region, describeRDSInstancesInput{
		PageNumber: pageNumber,
		PageSize:   pageSize,
	}, &out)
	return out, err
}

func (c *Client) DescribeRDSPostgreSQLInstances(ctx context.Context, region string, pageNumber, pageSize int32) (DescribeRDSPostgreSQLInstancesResponse, error) {
	var out DescribeRDSPostgreSQLInstancesResponse
	err := c.doRDSAction(ctx, ServiceRDSPostgreSQL, "DescribeDBInstances", region, describeRDSInstancesInput{
		PageNumber: pageNumber,
		PageSize:   pageSize,
	}, &out)
	return out, err
}

func (c *Client) DescribeRDSSQLServerInstances(ctx context.Context, region string, pageNumber, pageSize int32) (DescribeRDSSQLServerInstancesResponse, error) {
	var out DescribeRDSSQLServerInstancesResponse
	err := c.doRDSAction(ctx, ServiceRDSMSSQL, "DescribeDBInstances", region, describeRDSInstancesInput{
		PageNumber: pageNumber,
		PageSize:   pageSize,
	}, &out)
	return out, err
}

type RDSDBAccount struct {
	AccountName     string `json:"AccountName"`
	AccountStatus   string `json:"AccountStatus"`
	AccountType     string `json:"AccountType"`
	AccountPrivileges string `json:"AccountPrivileges,omitempty"`
}

type DescribeRDSAccountsResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		Accounts []RDSDBAccount `json:"Accounts"`
		Total    int32          `json:"Total"`
	} `json:"Result"`
}

type CreateRDSAccountInput struct {
	InstanceID      string `json:"InstanceId"`
	AccountName     string `json:"AccountName"`
	AccountPassword string `json:"AccountPassword"`
	AccountType     string `json:"AccountType,omitempty"`
}

type CreateRDSAccountResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
}

type DeleteRDSAccountInput struct {
	InstanceID  string `json:"InstanceId"`
	AccountName string `json:"AccountName"`
}

type DeleteRDSAccountResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
}

type describeRDSAccountsInput struct {
	InstanceID string `json:"InstanceId"`
}

func (c *Client) DescribeRDSDBAccounts(ctx context.Context, service, region, instanceID string) (DescribeRDSAccountsResponse, error) {
	var out DescribeRDSAccountsResponse
	err := c.doRDSAction(ctx, service, "DescribeDBAccounts", region, describeRDSAccountsInput{InstanceID: instanceID}, &out)
	return out, err
}

func (c *Client) CreateRDSDBAccount(ctx context.Context, service, region, instanceID, name, password string) (CreateRDSAccountResponse, error) {
	var out CreateRDSAccountResponse
	err := c.doRDSAction(ctx, service, "CreateDBAccount", region, CreateRDSAccountInput{
		InstanceID:      instanceID,
		AccountName:     name,
		AccountPassword: password,
		AccountType:     "Normal",
	}, &out)
	return out, err
}

func (c *Client) DeleteRDSDBAccount(ctx context.Context, service, region, instanceID, name string) (DeleteRDSAccountResponse, error) {
	var out DeleteRDSAccountResponse
	err := c.doRDSAction(ctx, service, "DeleteDBAccount", region, DeleteRDSAccountInput{
		InstanceID:  instanceID,
		AccountName: name,
	}, &out)
	return out, err
}

func (c *Client) doRDSAction(ctx context.Context, service, action, region string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return c.DoOpenAPI(ctx, Request{
		Service:    service,
		Version:    rdsAPIVersion,
		Action:     action,
		Method:     http.MethodPost,
		Region:     region,
		Path:       "/",
		Body:       body,
		Idempotent: true,
	}, out)
}
