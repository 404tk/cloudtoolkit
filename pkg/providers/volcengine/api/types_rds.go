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
