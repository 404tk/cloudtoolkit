package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

const ecsAPIVersion = "2020-04-01"

type DescribeRegionsResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		NextToken string      `json:"NextToken"`
		Regions   []ECSRegion `json:"Regions"`
	} `json:"Result"`
}

type ECSRegion struct {
	RegionID string `json:"RegionId"`
}

type DescribeInstancesResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		NextToken string        `json:"NextToken"`
		Instances []ECSInstance `json:"Instances"`
	} `json:"Result"`
}

type ECSInstance struct {
	InstanceID        string                `json:"InstanceId"`
	Hostname          string                `json:"Hostname"`
	Status            string                `json:"Status"`
	OSType            string                `json:"OsType"`
	EipAddress        ECSEipAddress         `json:"EipAddress"`
	NetworkInterfaces []ECSNetworkInterface `json:"NetworkInterfaces"`
}

type ECSEipAddress struct {
	IPAddress string `json:"IpAddress"`
}

type ECSNetworkInterface struct {
	PrimaryIPAddress string `json:"PrimaryIpAddress"`
}

func (c *Client) DescribeRegions(ctx context.Context, region string, maxResults int32) (DescribeRegionsResponse, error) {
	query := url.Values{}
	query.Set("MaxResults", strconv.FormatInt(int64(maxResults), 10))
	var out DescribeRegionsResponse
	err := c.DoOpenAPI(ctx, Request{
		Service:    "ecs",
		Version:    ecsAPIVersion,
		Action:     "DescribeRegions",
		Method:     http.MethodGet,
		Region:     region,
		Path:       "/",
		Query:      query,
		Idempotent: true,
	}, &out)
	return out, err
}

func (c *Client) DescribeInstances(ctx context.Context, region string, maxResults int32, nextToken string) (DescribeInstancesResponse, error) {
	query := url.Values{}
	query.Set("MaxResults", strconv.FormatInt(int64(maxResults), 10))
	query.Set("NextToken", nextToken)
	var out DescribeInstancesResponse
	err := c.DoOpenAPI(ctx, Request{
		Service:    "ecs",
		Version:    ecsAPIVersion,
		Action:     "DescribeInstances",
		Method:     http.MethodGet,
		Region:     region,
		Path:       "/",
		Query:      query,
		Idempotent: true,
	}, &out)
	return out, err
}
