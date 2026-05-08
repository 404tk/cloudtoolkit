package api

// JDCloud AuditTrail lookupEvents. This read-only API backs event-check dump in
// authorized environments.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ActionTrailEvent struct {
	EventTime           ActionTrailTimestamp  `json:"eventTime"`
	EventVersion        string                `json:"eventVersion"`
	Service             string                `json:"service"`
	ServiceAPIVersion   string                `json:"serviceApiVersion"`
	EventName           string                `json:"eventName"`
	EventSource         string                `json:"eventSource"`
	EventID             string                `json:"eventId"`
	EventType           string                `json:"eventType"`
	Region              string                `json:"region"`
	IP                  string                `json:"ip"`
	UserAgent           string                `json:"userAgent"`
	ErrorCode           string                `json:"errorCode"`
	ErrorMessage        string                `json:"errorMessage"`
	RequestID           string                `json:"requestId"`
	Plane               string                `json:"plane"`
	Classification      string                `json:"classification"`
	Account             string                `json:"account"`
	AccessKeyID         string                `json:"accessKeyId"`
	Resources           []ActionTrailResource `json:"resources"`
	Identity            ActionTrailIdentity   `json:"identity"`
	AccountGroup        string                `json:"accountGroup"`
	Request             string                `json:"request"`
	Response            string                `json:"response"`
	AdditionalEventData string                `json:"additionalEventData"`
}

type ActionTrailResource struct {
	ResourceName string `json:"resourceName"`
	ResourceID   string `json:"resourceId"`
	ResourceType string `json:"resourceType"`
	Name         string `json:"name"`
	ID           string `json:"id"`
	Type         string `json:"type"`
}

type ActionTrailIdentity struct {
	Type              string `json:"type"`
	Principal         string `json:"principal"`
	ERPPrincipal      string `json:"erpPrincipal"`
	Account           string `json:"account"`
	PreviousPrincipal string `json:"previousPrincipal"`
	InvokedBy         string `json:"invokedBy"`
	MFA               string `json:"mfa"`
}

type DescribeActionTrailEventsResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		PageSize    int                `json:"pageSize"`
		PageNumber  int                `json:"pageNumber"`
		TotalNumber int64              `json:"totalNumber"`
		Events      []ActionTrailEvent `json:"events"`
	} `json:"result"`
}

// DescribeActionTrailEvents calls JDCloud AuditTrail lookupEvents:
// POST /v1/regions/{regionId}/events.
func (c *Client) DescribeActionTrailEvents(ctx context.Context, region string, start, end int64, pageNumber, pageSize int, lookupAttributes string) (DescribeActionTrailEventsResponse, error) {
	if region == "" || region == "all" {
		region = "cn-north-1"
	}
	body := actionTrailLookupEventsRequest{
		RegionID:   region,
		PageSize:   intPtr(pageSize),
		PageNumber: intPtr(pageNumber),
	}
	if start > 0 {
		body.StartTime = &start
	}
	if end > 0 {
		body.EndTime = &end
	}
	if lookupAttributes = strings.TrimSpace(lookupAttributes); lookupAttributes != "" {
		body.LookupAttributes = &lookupAttributes
	}
	rawBody, err := json.Marshal(body)
	if err != nil {
		return DescribeActionTrailEventsResponse{}, err
	}
	var resp DescribeActionTrailEventsResponse
	err = c.DoJSON(ctx, Request{
		Service: "audittrail",
		Region:  region,
		Method:  http.MethodPost,
		Version: "v1",
		Path:    "/regions/" + region + "/events",
		Body:    rawBody,
	}, &resp)
	return resp, err
}

type actionTrailLookupEventsRequest struct {
	RegionID         string  `json:"regionId"`
	StartTime        *int64  `json:"startTime,omitempty"`
	EndTime          *int64  `json:"endTime,omitempty"`
	Classification   *string `json:"classification,omitempty"`
	PageSize         *int    `json:"pageSize,omitempty"`
	PageNumber       *int    `json:"pageNumber,omitempty"`
	LookupAttributes *string `json:"lookupAttributes,omitempty"`
}

func intPtr(value int) *int {
	if value <= 0 {
		return nil
	}
	return &value
}

type ActionTrailTimestamp int64

func (t *ActionTrailTimestamp) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}
	if trimmed[0] == '"' {
		var text string
		if err := json.Unmarshal(trimmed, &text); err != nil {
			return err
		}
		text = strings.TrimSpace(text)
		if text == "" {
			return nil
		}
		if value, err := strconv.ParseInt(text, 10, 64); err == nil {
			*t = ActionTrailTimestamp(value)
			return nil
		}
		parsed, err := time.Parse(time.RFC3339, text)
		if err != nil {
			return fmt.Errorf("jdcloud audittrail: invalid eventTime %q", text)
		}
		*t = ActionTrailTimestamp(parsed.Unix())
		return nil
	}
	var value int64
	if err := json.Unmarshal(trimmed, &value); err != nil {
		return err
	}
	*t = ActionTrailTimestamp(value)
	return nil
}
