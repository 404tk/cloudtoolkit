package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

type DescribeSASSuspEventsResponse struct {
	CurrentPage int            `json:"CurrentPage"`
	PageSize    int            `json:"PageSize"`
	RequestID   string         `json:"RequestId"`
	TotalCount  int            `json:"TotalCount"`
	Count       int            `json:"Count"`
	SuspEvents  []SASSuspEvent `json:"SuspEvents"`
}

type SASSuspEvent struct {
	SecurityEventIDs      string           `json:"SecurityEventIds"`
	AlarmEventNameDisplay string           `json:"AlarmEventNameDisplay"`
	InstanceName          string           `json:"InstanceName"`
	EventStatus           int              `json:"EventStatus"`
	LastTime              string           `json:"LastTime"`
	Details               []SASEventDetail `json:"Details"`
}

type SASEventDetail struct {
	NameDisplay  string `json:"NameDisplay"`
	ValueDisplay string `json:"ValueDisplay"`
}

func (c *Client) DescribeSASSuspEvents(ctx context.Context, region string) (DescribeSASSuspEventsResponse, error) {
	var resp DescribeSASSuspEventsResponse
	err := c.Do(ctx, Request{
		Product:    "Sas",
		Version:    "2018-12-03",
		Action:     "DescribeSuspEvents",
		Region:     region,
		Method:     http.MethodPost,
		Idempotent: true,
	}, &resp)
	return resp, err
}

type HandleSASSecurityEventsResponse struct {
	RequestID                    string                              `json:"RequestId"`
	HandleSecurityEventsResponse HandleSASSecurityEventsResponseItem `json:"HandleSecurityEventsResponse"`
}

type HandleSASSecurityEventsResponseItem struct {
	TaskID int64 `json:"TaskId"`
}

func (c *Client) HandleSASSecurityEvents(ctx context.Context, region, operationCode string, securityEventIDs []string) (HandleSASSecurityEventsResponse, error) {
	query := url.Values{}
	query.Set("OperationCode", operationCode)
	query.Set("MarkBatch", "true")
	for i, id := range securityEventIDs {
		query.Set("SecurityEventIds."+strconv.Itoa(i+1), id)
	}

	var resp HandleSASSecurityEventsResponse
	err := c.Do(ctx, Request{
		Product: "Sas",
		Version: "2018-12-03",
		Action:  "HandleSecurityEvents",
		Region:  region,
		Method:  http.MethodPost,
		Query:   query,
	}, &resp)
	return resp, err
}
