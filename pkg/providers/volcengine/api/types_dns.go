package api

import (
	"context"
	"encoding/json"
	"net/http"
)

const dnsAPIVersion = "2018-08-01"

type ListDNSZonesResponse struct {
	Total int32     `json:"Total"`
	Zones []DNSZone `json:"Zones"`
}

type DNSZone struct {
	ZID      int64  `json:"ZID"`
	ZoneName string `json:"ZoneName"`
}

type ListDNSRecordsResponse struct {
	PageNumber int32       `json:"PageNumber"`
	PageSize   int32       `json:"PageSize"`
	Records    []DNSRecord `json:"Records"`
	TotalCount int32       `json:"TotalCount"`
}

type DNSRecord struct {
	Enable *bool  `json:"Enable,omitempty"`
	FQDN   string `json:"FQDN"`
	Host   string `json:"Host"`
	Type   string `json:"Type"`
	Value  string `json:"Value"`
}

type listDNSZonesInput struct {
	PageNumber int32 `json:"PageNumber"`
	PageSize   int32 `json:"PageSize"`
}

type listDNSRecordsInput struct {
	PageNumber int32 `json:"PageNumber"`
	PageSize   int32 `json:"PageSize"`
	ZID        int64 `json:"ZID"`
}

func (c *Client) ListDNSZones(ctx context.Context, pageNumber, pageSize int32) (ListDNSZonesResponse, error) {
	var out ListDNSZonesResponse
	err := c.doDNSAction(ctx, "ListZones", listDNSZonesInput{
		PageNumber: pageNumber,
		PageSize:   pageSize,
	}, &out)
	return out, err
}

func (c *Client) ListDNSRecords(ctx context.Context, zid int64, pageNumber, pageSize int32) (ListDNSRecordsResponse, error) {
	var out ListDNSRecordsResponse
	err := c.doDNSAction(ctx, "ListRecords", listDNSRecordsInput{
		PageNumber: pageNumber,
		PageSize:   pageSize,
		ZID:        zid,
	}, &out)
	return out, err
}

func (c *Client) doDNSAction(ctx context.Context, action string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return c.DoOpenAPI(ctx, Request{
		Service:    "dns",
		Version:    dnsAPIVersion,
		Action:     action,
		Method:     http.MethodPost,
		Path:       "/",
		Body:       body,
		Idempotent: true,
	}, out)
}
