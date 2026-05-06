package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

const auditAPIVersion = "2021-02-23"

// AuditEvent maps the per-event fields returned by the Volcengine Audit
// `DescribeEvents` action. Only fields useful for the validation flow are
// projected.
type AuditEvent struct {
	EventID        string `json:"EventId"`
	EventName      string `json:"EventName"`
	EventTime      string `json:"EventTime"`
	EventSource    string `json:"EventSource"`
	UserIdentity   string `json:"UserIdentity"`
	SourceIPAddress string `json:"SourceIPAddress"`
	Region         string `json:"Region"`
	Status         string `json:"Status"`
	AccessKeyID    string `json:"AccessKeyId"`
	ResourceName   string `json:"ResourceName"`
	ResourceType   string `json:"ResourceType"`
}

type DescribeEventsResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		Events    []AuditEvent `json:"Events"`
		PageToken string       `json:"PageToken"`
	} `json:"Result"`
}

// DescribeAuditEvents calls the Volcengine Audit DescribeEvents action.
// `start` and `end` are unix seconds (0 = unset, fall back to service default).
func (c *Client) DescribeAuditEvents(ctx context.Context, region string, start, end int64, pageSize int, pageToken string) (DescribeEventsResponse, error) {
	query := url.Values{}
	if start > 0 {
		query.Set("StartTime", strconv.FormatInt(start, 10))
	}
	if end > 0 {
		query.Set("EndTime", strconv.FormatInt(end, 10))
	}
	if pageSize > 0 {
		query.Set("PageSize", strconv.Itoa(pageSize))
	}
	if pageToken != "" {
		query.Set("PageToken", pageToken)
	}
	var out DescribeEventsResponse
	err := c.DoOpenAPI(ctx, Request{
		Service:    "audit",
		Version:    auditAPIVersion,
		Action:     "DescribeEvents",
		Method:     http.MethodGet,
		Region:     region,
		Path:       "/",
		Query:      query,
		Idempotent: true,
	}, &out)
	return out, err
}
