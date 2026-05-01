package api

import (
	"context"
	"encoding/json"
	"net/http"
)

// AWS CloudTrail LookupEvents — JSON-1.1 RPC like SSM.
const (
	cloudTrailContentType    = "application/x-amz-json-1.1"
	cloudTrailLookupEventsTg = "com.amazonaws.cloudtrail.v20131101.CloudTrail_20131101.LookupEvents"
)

type LookupEventsInput struct {
	StartTime        *float64           `json:"StartTime,omitempty"`
	EndTime          *float64           `json:"EndTime,omitempty"`
	MaxResults       *int64             `json:"MaxResults,omitempty"`
	NextToken        *string            `json:"NextToken,omitempty"`
	LookupAttributes []LookupAttribute  `json:"LookupAttributes,omitempty"`
}

type LookupAttribute struct {
	AttributeKey   string `json:"AttributeKey"`
	AttributeValue string `json:"AttributeValue"`
}

type LookupEventsOutput struct {
	NextToken string         `json:"NextToken"`
	Events    []CloudTrailEvent `json:"Events"`
}

type CloudTrailEvent struct {
	EventID         string                 `json:"EventId"`
	EventName       string                 `json:"EventName"`
	EventTime       float64                `json:"EventTime"`
	EventSource     string                 `json:"EventSource"`
	Username        string                 `json:"Username"`
	AccessKeyID     string                 `json:"AccessKeyId"`
	Resources       []CloudTrailResource   `json:"Resources"`
	CloudTrailEvent string                 `json:"CloudTrailEvent"`
}

type CloudTrailResource struct {
	ResourceType string `json:"ResourceType"`
	ResourceName string `json:"ResourceName"`
}

// CloudTrailLookupEvents reads recent management-event entries from AWS
// CloudTrail. startTime / endTime are Unix seconds (0 = unset → CloudTrail
// default 90-day window). nextToken paginates. The response includes both
// the parsed event header *and* the original `CloudTrailEvent` JSON blob,
// which the caller can re-parse for richer fields.
func (c *Client) CloudTrailLookupEvents(ctx context.Context, region string, startTime, endTime int64, maxResults int64, nextToken string) (LookupEventsOutput, error) {
	input := LookupEventsInput{}
	if startTime > 0 {
		v := float64(startTime)
		input.StartTime = &v
	}
	if endTime > 0 {
		v := float64(endTime)
		input.EndTime = &v
	}
	if maxResults > 0 {
		v := maxResults
		input.MaxResults = &v
	}
	if nextToken != "" {
		t := nextToken
		input.NextToken = &t
	}
	body, err := json.Marshal(input)
	if err != nil {
		return LookupEventsOutput{}, err
	}
	headers := http.Header{}
	headers.Set("Content-Type", cloudTrailContentType)
	headers.Set("X-Amz-Target", cloudTrailLookupEventsTg)
	var out LookupEventsOutput
	err = c.DoRESTJSON(ctx, Request{
		Service:    "cloudtrail",
		Region:     region,
		Method:     http.MethodPost,
		Path:       "/",
		Body:       body,
		Headers:    headers,
		Idempotent: true,
	}, &out)
	return out, err
}
