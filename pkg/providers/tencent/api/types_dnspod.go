package api

import "context"

const defaultDNSListLimit = 3000

type DescribeDomainListRequest struct {
	Offset int64 `json:"Offset,omitempty"`
	Limit  int64 `json:"Limit,omitempty"`
}

type DescribeDomainListResponse struct {
	Response struct {
		DomainCountInfo struct {
			DomainTotal *uint64 `json:"DomainTotal"`
		} `json:"DomainCountInfo"`
		DomainList []DomainListItem `json:"DomainList"`
		RequestID  string           `json:"RequestId"`
	} `json:"Response"`
}

type DomainListItem struct {
	Name      *string `json:"Name"`
	Status    *string `json:"Status"`
	DNSStatus *string `json:"DNSStatus"`
}

func (c *Client) DescribeDomainList(ctx context.Context, region string, offset, limit int64) (DescribeDomainListResponse, error) {
	if limit <= 0 {
		limit = defaultDNSListLimit
	}
	var resp DescribeDomainListResponse
	err := c.DoJSON(
		ctx,
		"dnspod",
		"2021-03-23",
		"DescribeDomainList",
		normalizeRegion(region),
		DescribeDomainListRequest{
			Offset: offset,
			Limit:  limit,
		},
		&resp,
	)
	return resp, err
}

type DescribeRecordListRequest struct {
	Domain       string `json:"Domain"`
	Offset       uint64 `json:"Offset,omitempty"`
	Limit        uint64 `json:"Limit,omitempty"`
	ErrorOnEmpty string `json:"ErrorOnEmpty,omitempty"`
}

type DescribeRecordListResponse struct {
	Response struct {
		RecordCountInfo struct {
			TotalCount *uint64 `json:"TotalCount"`
			ListCount  *uint64 `json:"ListCount"`
		} `json:"RecordCountInfo"`
		RecordList []RecordListItem `json:"RecordList"`
		RequestID  string           `json:"RequestId"`
	} `json:"Response"`
}

type RecordListItem struct {
	Name   *string `json:"Name"`
	Type   *string `json:"Type"`
	Value  *string `json:"Value"`
	Status *string `json:"Status"`
}

func (c *Client) DescribeRecordList(ctx context.Context, region, domain string, offset, limit uint64) (DescribeRecordListResponse, error) {
	if limit == 0 {
		limit = defaultDNSListLimit
	}
	var resp DescribeRecordListResponse
	err := c.DoJSON(
		ctx,
		"dnspod",
		"2021-03-23",
		"DescribeRecordList",
		normalizeRegion(region),
		DescribeRecordListRequest{
			Domain:       domain,
			Offset:       offset,
			Limit:        limit,
			ErrorOnEmpty: "no",
		},
		&resp,
	)
	return resp, err
}
