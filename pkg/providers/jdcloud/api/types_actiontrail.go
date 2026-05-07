package api

// JDCloud ActionTrail (audit). The action and shape follow the same v1
// path-with-version family as other JDCloud REST services and are exercised by
// the event-check replay and focused tests.

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

type ActionTrailEvent struct {
	EventID         string `json:"eventId"`
	EventName       string `json:"eventName"`
	EventTime       string `json:"eventTime"`
	EventSource     string `json:"eventSource"`
	UserName        string `json:"userName"`
	SourceIPAddress string `json:"sourceIpAddress"`
	Region          string `json:"region"`
	Status          string `json:"status"`
	AccessKey       string `json:"accessKeyId"`
	ResourceName    string `json:"resourceName"`
	ResourceType    string `json:"resourceType"`
}

type DescribeActionTrailEventsResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		Events     []ActionTrailEvent `json:"events"`
		NextToken  string             `json:"nextToken,omitempty"`
		TotalCount int                `json:"totalCount,omitempty"`
	} `json:"result"`
}

// DescribeActionTrailEvents calls JDCloud ActionTrail to list recent audit
// events. The path mirrors other JDCloud regional list endpoints
// (`/v1/regions/<region>/<resource>:<action>` pattern).
func (c *Client) DescribeActionTrailEvents(ctx context.Context, region string, start, end int64, maxResults int, nextToken string) (DescribeActionTrailEventsResponse, error) {
	if region == "" || region == "all" {
		region = "cn-north-1"
	}
	query := url.Values{}
	if start > 0 {
		query.Set("startTime", strconv.FormatInt(start, 10))
	}
	if end > 0 {
		query.Set("endTime", strconv.FormatInt(end, 10))
	}
	if maxResults > 0 {
		query.Set("maxResults", strconv.Itoa(maxResults))
	}
	if nextToken != "" {
		query.Set("nextToken", nextToken)
	}
	var resp DescribeActionTrailEventsResponse
	err := c.DoJSON(ctx, Request{
		Service: "actiontrail",
		Region:  "",
		Method:  http.MethodGet,
		Version: "v1",
		Path:    "/regions/" + region + "/events:lookup",
		Query:   query,
	}, &resp)
	return resp, err
}
