package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

const defaultPageSize = 100

type GetCallerIdentityResponse struct {
	IdentityType string `json:"IdentityType"`
	AccountID    string `json:"AccountId"`
	RequestID    string `json:"RequestId"`
	PrincipalID  string `json:"PrincipalId"`
	UserID       string `json:"UserId"`
	Arn          string `json:"Arn"`
	RoleID       string `json:"RoleId"`
}

func (c *Client) GetCallerIdentity(ctx context.Context, region string) (GetCallerIdentityResponse, error) {
	var resp GetCallerIdentityResponse
	err := c.Do(ctx, Request{
		Product:    "Sts",
		Version:    "2015-04-01",
		Action:     "GetCallerIdentity",
		Region:     region,
		Method:     http.MethodPost,
		Idempotent: true,
	}, &resp)
	return resp, err
}

type QueryAccountBalanceResponse struct {
	Code      string             `json:"Code"`
	Message   string             `json:"Message"`
	RequestID string             `json:"RequestId"`
	Success   bool               `json:"Success"`
	Data      AccountBalanceData `json:"Data"`
}

type AccountBalanceData struct {
	AvailableCashAmount string `json:"AvailableCashAmount"`
}

func (c *Client) QueryAccountBalance(ctx context.Context, region string) (QueryAccountBalanceResponse, error) {
	var resp QueryAccountBalanceResponse
	err := c.Do(ctx, Request{
		Product:    "BssOpenApi",
		Version:    "2017-12-14",
		Action:     "QueryAccountBalance",
		Region:     region,
		Method:     http.MethodPost,
		Idempotent: true,
	}, &resp)
	return resp, err
}

type DescribeDomainsResponse struct {
	TotalCount int        `json:"TotalCount"`
	PageSize   int        `json:"PageSize"`
	RequestID  string     `json:"RequestId"`
	PageNumber int        `json:"PageNumber"`
	Domains    DomainList `json:"Domains"`
}

type DomainList struct {
	Domain []DomainSummary `json:"Domain"`
}

type DomainSummary struct {
	DomainName string `json:"DomainName"`
}

func (c *Client) DescribeDomains(ctx context.Context, region string, pageNumber, pageSize int) (DescribeDomainsResponse, error) {
	var resp DescribeDomainsResponse
	err := c.Do(ctx, Request{
		Product:    "Alidns",
		Version:    "2015-01-09",
		Action:     "DescribeDomains",
		Region:     region,
		Method:     http.MethodPost,
		Query:      pagingQuery(pageNumber, pageSize),
		Idempotent: true,
	}, &resp)
	return resp, err
}

type DescribeDomainRecordsResponse struct {
	TotalCount    int              `json:"TotalCount"`
	PageSize      int              `json:"PageSize"`
	RequestID     string           `json:"RequestId"`
	PageNumber    int              `json:"PageNumber"`
	DomainRecords DomainRecordList `json:"DomainRecords"`
}

type DomainRecordList struct {
	Record []DomainRecord `json:"Record"`
}

type DomainRecord struct {
	RR     string `json:"RR"`
	Type   string `json:"Type"`
	Value  string `json:"Value"`
	Status string `json:"Status"`
}

func (c *Client) DescribeDomainRecords(ctx context.Context, region, domainName string, pageNumber, pageSize int) (DescribeDomainRecordsResponse, error) {
	query := pagingQuery(pageNumber, pageSize)
	query.Set("DomainName", domainName)

	var resp DescribeDomainRecordsResponse
	err := c.Do(ctx, Request{
		Product:    "Alidns",
		Version:    "2015-01-09",
		Action:     "DescribeDomainRecords",
		Region:     region,
		Method:     http.MethodPost,
		Query:      query,
		Idempotent: true,
	}, &resp)
	return resp, err
}

func pagingQuery(pageNumber, pageSize int) url.Values {
	if pageNumber <= 0 {
		pageNumber = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	query := url.Values{}
	query.Set("PageNumber", strconv.Itoa(pageNumber))
	query.Set("PageSize", strconv.Itoa(pageSize))
	return query
}
