package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

type CreateCommandResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		CommandID string `json:"CommandId"`
	} `json:"Result"`
}

type InvokeCommandResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		InvocationID string `json:"InvocationId"`
	} `json:"Result"`
}

type DeleteCommandResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		CommandID string `json:"CommandId"`
	} `json:"Result"`
}

type DescribeCloudAssistantStatusResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		Instances  []ECSCloudAssistantInstance `json:"Instances"`
		PageNumber int32                       `json:"PageNumber"`
		PageSize   int32                       `json:"PageSize"`
		TotalCount int32                       `json:"TotalCount"`
	} `json:"Result"`
}

type ECSCloudAssistantInstance struct {
	ClientVersion     string `json:"ClientVersion"`
	HostName          string `json:"HostName"`
	InstanceID        string `json:"InstanceId"`
	InstanceName      string `json:"InstanceName"`
	LastHeartbeatTime string `json:"LastHeartbeatTime"`
	OSType            string `json:"OsType"`
	OSVersion         string `json:"OsVersion"`
	Status            string `json:"Status"`
}

type DescribeInvocationResultsResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		InvocationResults []ECSInvocationResult `json:"InvocationResults"`
		PageNumber        int32                 `json:"PageNumber"`
		PageSize          int32                 `json:"PageSize"`
		TotalCount        int32                 `json:"TotalCount"`
	} `json:"Result"`
}

type ECSInvocationResult struct {
	CommandID              string `json:"CommandId"`
	InvocationID           string `json:"InvocationId"`
	InstanceID             string `json:"InstanceId"`
	InvocationStatus       string `json:"InvocationStatus"`
	InvocationResultStatus string `json:"InvocationResultStatus"`
	Output                 string `json:"Output"`
	ErrorInfo              string `json:"ErrorInfo"`
	ErrorMessage           string `json:"ErrorMessage"`
	ErrorCode              string `json:"ErrorCode"`
}

func (r ECSInvocationResult) Status() string {
	if status := strings.TrimSpace(r.InvocationStatus); status != "" {
		return status
	}
	return strings.TrimSpace(r.InvocationResultStatus)
}

func (r ECSInvocationResult) Message() string {
	if msg := strings.TrimSpace(r.ErrorMessage); msg != "" {
		return msg
	}
	return strings.TrimSpace(r.ErrorInfo)
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

func (c *Client) CreateCommand(ctx context.Context, region, name, commandType, commandContent, contentEncoding string) (CreateCommandResponse, error) {
	query := url.Values{}
	query.Set("Name", strings.TrimSpace(name))
	query.Set("Type", strings.TrimSpace(commandType))
	query.Set("CommandContent", commandContent)
	if encoding := strings.TrimSpace(contentEncoding); encoding != "" {
		query.Set("ContentEncoding", encoding)
	}
	var out CreateCommandResponse
	err := c.DoOpenAPI(ctx, Request{
		Service:    "ecs",
		Version:    ecsAPIVersion,
		Action:     "CreateCommand",
		Method:     http.MethodGet,
		Region:     region,
		Path:       "/",
		Query:      query,
		Idempotent: true,
	}, &out)
	return out, err
}

func (c *Client) InvokeCommand(ctx context.Context, region, commandID, invocationName string, instanceIDs []string) (InvokeCommandResponse, error) {
	query := url.Values{}
	query.Set("CommandId", strings.TrimSpace(commandID))
	query.Set("InvocationName", strings.TrimSpace(invocationName))
	for i, instanceID := range instanceIDs {
		instanceID = strings.TrimSpace(instanceID)
		if instanceID == "" {
			continue
		}
		query.Set("InstanceIds."+strconv.Itoa(i+1), instanceID)
	}
	var out InvokeCommandResponse
	err := c.DoOpenAPI(ctx, Request{
		Service:    "ecs",
		Version:    ecsAPIVersion,
		Action:     "InvokeCommand",
		Method:     http.MethodGet,
		Region:     region,
		Path:       "/",
		Query:      query,
		Idempotent: true,
	}, &out)
	return out, err
}

func (c *Client) DeleteCommand(ctx context.Context, region, commandID string) (DeleteCommandResponse, error) {
	query := url.Values{}
	query.Set("CommandId", strings.TrimSpace(commandID))
	var out DeleteCommandResponse
	err := c.DoOpenAPI(ctx, Request{
		Service:    "ecs",
		Version:    ecsAPIVersion,
		Action:     "DeleteCommand",
		Method:     http.MethodGet,
		Region:     region,
		Path:       "/",
		Query:      query,
		Idempotent: true,
	}, &out)
	return out, err
}

func (c *Client) DescribeCloudAssistantStatus(ctx context.Context, region string, instanceIDs []string, osType string, pageSize int32) (DescribeCloudAssistantStatusResponse, error) {
	query := url.Values{}
	for i, instanceID := range instanceIDs {
		instanceID = strings.TrimSpace(instanceID)
		if instanceID == "" {
			continue
		}
		query.Set("InstanceIds."+strconv.Itoa(i+1), instanceID)
	}
	if osType = strings.TrimSpace(osType); osType != "" {
		query.Set("OSType", osType)
	}
	query.Set("PageNumber", "1")
	if pageSize > 0 {
		query.Set("PageSize", strconv.FormatInt(int64(pageSize), 10))
	}

	var out DescribeCloudAssistantStatusResponse
	err := c.DoOpenAPI(ctx, Request{
		Service:    "ecs",
		Version:    ecsAPIVersion,
		Action:     "DescribeCloudAssistantStatus",
		Method:     http.MethodGet,
		Region:     region,
		Path:       "/",
		Query:      query,
		Idempotent: true,
	}, &out)
	return out, err
}

func (c *Client) DescribeInvocationResults(ctx context.Context, region, invocationID, commandID, instanceID string, maxResults int32) (DescribeInvocationResultsResponse, error) {
	query := url.Values{}
	if invocationID = strings.TrimSpace(invocationID); invocationID != "" {
		query.Set("InvocationId", invocationID)
	}
	if commandID = strings.TrimSpace(commandID); commandID != "" {
		query.Set("CommandId", commandID)
	}
	if instanceID = strings.TrimSpace(instanceID); instanceID != "" {
		query.Set("InstanceId", instanceID)
	}
	if maxResults > 0 {
		query.Set("PageNumber", "1")
		query.Set("PageSize", strconv.FormatInt(int64(maxResults), 10))
	}
	var out DescribeInvocationResultsResponse
	err := c.DoOpenAPI(ctx, Request{
		Service:    "ecs",
		Version:    ecsAPIVersion,
		Action:     "DescribeInvocationResults",
		Method:     http.MethodGet,
		Region:     region,
		Path:       "/",
		Query:      query,
		Idempotent: true,
	}, &out)
	return out, err
}
