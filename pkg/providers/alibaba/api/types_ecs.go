package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

type DescribeECSRegionsResponse struct {
	RequestID string        `json:"RequestId"`
	Regions   ECSRegionList `json:"Regions"`
}

type ECSRegionList struct {
	Region []ECSRegion `json:"Region"`
}

type ECSRegion struct {
	RegionID string `json:"RegionId"`
}

func (c *Client) DescribeECSRegions(ctx context.Context, region string) (DescribeECSRegionsResponse, error) {
	var resp DescribeECSRegionsResponse
	err := c.Do(ctx, Request{
		Product:    "Ecs",
		Version:    "2014-05-26",
		Action:     "DescribeRegions",
		Region:     region,
		Method:     http.MethodPost,
		Idempotent: true,
	}, &resp)
	return resp, err
}

type DescribeECSInstancesResponse struct {
	PageSize   int             `json:"PageSize"`
	PageNumber int             `json:"PageNumber"`
	RequestID  string          `json:"RequestId"`
	TotalCount int             `json:"TotalCount"`
	Instances  ECSInstanceList `json:"Instances"`
}

type ECSInstanceList struct {
	Instance []ECSInstance `json:"Instance"`
}

type ECSInstance struct {
	HostName          string               `json:"HostName"`
	InstanceID        string               `json:"InstanceId"`
	OSType            string               `json:"OSType"`
	PublicIP          ECSPublicIPList      `json:"PublicIpAddress"`
	NetworkInterfaces ECSNetworkInterfaces `json:"NetworkInterfaces"`
	EIPAddress        ECSEIPAddress        `json:"EipAddress"`
}

type ECSPublicIPList struct {
	IPAddress []string `json:"IpAddress"`
}

type ECSNetworkInterfaces struct {
	NetworkInterface []ECSNetworkInterface `json:"NetworkInterface"`
}

type ECSNetworkInterface struct {
	PrimaryIPAddress string           `json:"PrimaryIpAddress"`
	PrivateIPSets    ECSPrivateIPSets `json:"PrivateIpSets"`
}

type ECSPrivateIPSets struct {
	PrivateIPSet []ECSPrivateIPSet `json:"PrivateIpSet"`
}

type ECSPrivateIPSet struct {
	PrivateIPAddress string `json:"PrivateIpAddress"`
}

type ECSEIPAddress struct {
	IPAddress string `json:"IpAddress"`
}

func (c *Client) DescribeECSInstances(ctx context.Context, region string, pageNumber, pageSize int) (DescribeECSInstancesResponse, error) {
	var resp DescribeECSInstancesResponse
	err := c.Do(ctx, Request{
		Product:    "Ecs",
		Version:    "2014-05-26",
		Action:     "DescribeInstances",
		Region:     region,
		Method:     http.MethodPost,
		Query:      pagingQuery(pageNumber, pageSize),
		Idempotent: true,
	}, &resp)
	return resp, err
}

type RunECSCommandResponse struct {
	RequestID string `json:"RequestId"`
	CommandID string `json:"CommandId"`
	InvokeID  string `json:"InvokeId"`
}

func (c *Client) RunECSCommand(ctx context.Context, region, commandType, commandContent, contentEncoding string, instanceIDs []string) (RunECSCommandResponse, error) {
	query := url.Values{}
	query.Set("Type", commandType)
	query.Set("CommandContent", commandContent)
	if contentEncoding != "" {
		query.Set("ContentEncoding", contentEncoding)
	}
	for i, instanceID := range instanceIDs {
		if instanceID == "" {
			continue
		}
		query.Set("InstanceId."+strconv.Itoa(i+1), instanceID)
	}

	var resp RunECSCommandResponse
	err := c.Do(ctx, Request{
		Product: "Ecs",
		Version: "2014-05-26",
		Action:  "RunCommand",
		Region:  region,
		Method:  http.MethodPost,
		Query:   query,
	}, &resp)
	return resp, err
}

type DescribeECSInvocationResultsResponse struct {
	RequestID  string        `json:"RequestId"`
	Invocation ECSInvocation `json:"Invocation"`
}

type ECSInvocation struct {
	CommandID         string               `json:"CommandId"`
	InvokeID          string               `json:"InvokeId"`
	InvocationResults ECSInvocationResults `json:"InvocationResults"`
}

type ECSInvocationResults struct {
	InvocationResult []ECSInvocationResult `json:"InvocationResult"`
}

type ECSInvocationResult struct {
	InvokeRecordStatus string `json:"InvokeRecordStatus"`
	Output             string `json:"Output"`
	ErrorInfo          string `json:"ErrorInfo"`
}

func (c *Client) DescribeECSInvocationResults(ctx context.Context, region, commandID string) (DescribeECSInvocationResultsResponse, error) {
	query := url.Values{}
	query.Set("CommandId", commandID)
	query.Set("ContentEncoding", "PlainText")
	query.Set("PageSize", "1")

	var resp DescribeECSInvocationResultsResponse
	err := c.Do(ctx, Request{
		Product:    "Ecs",
		Version:    "2014-05-26",
		Action:     "DescribeInvocationResults",
		Region:     region,
		Method:     http.MethodPost,
		Query:      query,
		Idempotent: true,
	}, &resp)
	return resp, err
}
