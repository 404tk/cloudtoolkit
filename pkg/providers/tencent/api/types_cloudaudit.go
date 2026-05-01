package api

import "context"

const cloudAuditVersion = "2019-03-19"

type LookUpEventsRequest struct {
	StartTime        *int64                 `json:"StartTime,omitempty"`
	EndTime          *int64                 `json:"EndTime,omitempty"`
	MaxResults       *uint64                `json:"MaxResults,omitempty"`
	NextToken        *string                `json:"NextToken,omitempty"`
	LookupAttributes []LookupAttribute      `json:"LookupAttributes,omitempty"`
}

type LookupAttribute struct {
	AttributeKey   *string `json:"AttributeKey,omitempty"`
	AttributeValue *string `json:"AttributeValue,omitempty"`
}

type LookUpEventsResponse struct {
	Response struct {
		NextToken *string             `json:"NextToken"`
		ListOver  *bool               `json:"ListOver"`
		Events    []CloudAuditEvent   `json:"Events"`
		RequestID string              `json:"RequestId"`
	} `json:"Response"`
}

type CloudAuditEvent struct {
	EventID         *string `json:"EventId"`
	EventName       *string `json:"EventName"`
	EventNameCn     *string `json:"EventNameCn"`
	EventTime       *string `json:"EventTime"`
	EventRegion     *string `json:"EventRegion"`
	Username        *string `json:"Username"`
	SourceIPAddress *string `json:"SourceIPAddress"`
	ResourceTypeCn  *string `json:"ResourceTypeCn"`
	ResourceName    *string `json:"ResourceName"`
	Status          *uint64 `json:"Status"`
	SecretID        *string `json:"SecretId"`
	APIVersion      *string `json:"ApiVersion"`
}

// LookUpEvents queries the Tencent CloudAudit operation log. The default
// caller behaviour mirrors the existing alibaba SAS dump: surface the most
// recent operations so a CSPM detection can be cross-referenced. StartTime /
// EndTime are unix seconds; pass 0 to leave them unset and fall back to the
// CloudAudit default lookback window.
func (c *Client) LookUpEvents(ctx context.Context, region string, startTime, endTime int64, maxResults uint64, nextToken string) (LookUpEventsResponse, error) {
	req := LookUpEventsRequest{}
	if startTime > 0 {
		ts := startTime
		req.StartTime = &ts
	}
	if endTime > 0 {
		te := endTime
		req.EndTime = &te
	}
	if maxResults > 0 {
		req.MaxResults = uint64Ptr(maxResults)
	}
	if nt := nextToken; nt != "" {
		req.NextToken = &nt
	}
	var resp LookUpEventsResponse
	err := c.DoJSON(ctx, "cloudaudit", cloudAuditVersion, "LookUpEvents", region, req, &resp)
	return resp, err
}
