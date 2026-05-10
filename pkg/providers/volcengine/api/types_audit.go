package api

import (
	"context"
	"encoding/json"
	"net/http"
)

const (
	cloudTrailAPIVersion = "2021-09-01"
	cloudTrailService    = "cloudtrail"
	cloudTrailSignName   = "cloud_trail"
)

// AuditEvent maps the per-event fields returned by Volcengine CloudTrail
// `LookupEvents`. Only fields useful for the validation flow are
// projected.
type AuditEvent struct {
	AccessKeyID        string                 `json:"AccessKeyID"`
	ErrorCode          string                 `json:"ErrorCode"`
	EventDetail        string                 `json:"EventDetail"`
	EventID            string                 `json:"EventID"`
	EventName          string                 `json:"EventName"`
	EventNameDisplay   string                 `json:"EventNameDisplay"`
	EventSource        string                 `json:"EventSource"`
	EventSourceDisplay string                 `json:"EventSourceDisplay"`
	EventTime          string                 `json:"EventTime"`
	Region             string                 `json:"Region"`
	RelatedResources   []AuditRelatedResource `json:"RelatedResources"`
	RequestID          string                 `json:"RequestID"`
	SourceIPAddress    string                 `json:"SourceIPAddress"`
	UserName           string                 `json:"UserName"`
}

type AuditRelatedResource struct {
	IntegratedTrn       string `json:"IntegratedTrn"`
	ResourceID          string `json:"ResourceID"`
	ResourceType        string `json:"ResourceType"`
	ResourceTypeDisplay string `json:"ResourceTypeDisplay"`
	ServiceCode         string `json:"ServiceCode"`
	SourceType          string `json:"SourceType"`
}

type LookupEventsResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		NextToken string       `json:"NextToken"`
		Trails    []AuditEvent `json:"Trails"`
	} `json:"Result"`
}

type lookupEventsInput struct {
	StartTime        *int64                                 `json:"StartTime,omitempty"`
	EndTime          *int64                                 `json:"EndTime,omitempty"`
	MaxResults       *int32                                 `json:"MaxResults,omitempty"`
	NextToken        *string                                `json:"NextToken,omitempty"`
	LookupConditions []*LookupConditionForLookupEventsInput `json:"LookupConditions,omitempty"`
}

type LookupConditionForLookupEventsInput struct {
	LookupConditionKey   *string `json:"LookupConditionKey,omitempty"`
	LookupConditionValue *string `json:"LookupConditionValue,omitempty"`
}

// LookupAuditEvents calls the Volcengine CloudTrail LookupEvents action.
// `start` and `end` are unix seconds (0 = unset, fall back to service default).
func (c *Client) LookupAuditEvents(ctx context.Context, region string, start, end int64, pageSize int, nextToken, accessKey string) (LookupEventsResponse, error) {
	input := lookupEventsInput{}
	if start > 0 {
		input.StartTime = &start
	}
	if end > 0 {
		input.EndTime = &end
	}
	if pageSize > 0 {
		value := int32(pageSize)
		input.MaxResults = &value
	}
	if nextToken != "" {
		input.NextToken = &nextToken
	}
	if accessKey != "" {
		input.LookupConditions = []*LookupConditionForLookupEventsInput{
			{
				LookupConditionKey:   stringPtr("AccessKeyID"),
				LookupConditionValue: stringPtr(accessKey),
			},
		}
	}
	body, err := json.Marshal(input)
	if err != nil {
		return LookupEventsResponse{}, err
	}
	var out LookupEventsResponse
	err = c.DoOpenAPI(ctx, Request{
		Service:     cloudTrailService,
		SignService: cloudTrailSignName,
		Version:     cloudTrailAPIVersion,
		Action:      "LookupEvents",
		Method:      http.MethodPost,
		Region:      region,
		Path:        "/",
		Body:        body,
		Idempotent:  true,
	}, &out)
	return out, err
}

func stringPtr(v string) *string {
	return &v
}
