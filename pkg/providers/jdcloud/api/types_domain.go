package api

// JDCloud domainservice — DescribeDomains + DescribeResourceRecord. The
// `domainservice` service is the newer Describe* family of the JDCloud DNS
// management API; URL conventions mirror the SDK at
//   github.com/jdcloud-api/jdcloud-sdk-go@v1.64.0/services/domainservice
// Verify against upstream before relying on this in production deployments.

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// DomainInfo is the JDCloud DNS domain shape projected for cloudlist.
type DomainInfo struct {
	ID              int      `json:"id"`
	DomainName      string   `json:"domainName"`
	CreateTime      int64    `json:"createTime"`
	ExpirationDate  int64    `json:"expirationDate"`
	PackID          int      `json:"packId"`
	PackName        string   `json:"packName"`
	ResolvingStatus string   `json:"resolvingStatus"`
	Creator         string   `json:"creator"`
	JcloudNs        bool     `json:"jcloudNs"`
	LockStatus      int      `json:"lockStatus"`
	ProbeNsList     []string `json:"probeNsList,omitempty"`
	DefNsList       []string `json:"defNsList,omitempty"`
}

type DescribeDomainsResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		DataList     []DomainInfo `json:"dataList"`
		CurrentCount int          `json:"currentCount"`
		TotalCount   int          `json:"totalCount"`
		TotalPage    int          `json:"totalPage"`
	} `json:"result"`
}

func (c *Client) DescribeDomains(ctx context.Context, region string, pageNumber, pageSize int) (DescribeDomainsResponse, error) {
	if region == "" || region == "all" {
		region = "cn-north-1"
	}
	query := url.Values{}
	if pageNumber > 0 {
		query.Set("pageNumber", strconv.Itoa(pageNumber))
	}
	if pageSize > 0 {
		query.Set("pageSize", strconv.Itoa(pageSize))
	}
	var resp DescribeDomainsResponse
	err := c.DoJSON(ctx, Request{
		Service:    "domainservice",
		Region:     region,
		Method:     http.MethodGet,
		Version:    "v2",
		Path:       "/regions/" + region + "/domain",
		Query:      query,
		Idempotent: true,
	}, &resp)
	return resp, err
}

// DomainResourceRecord is the JDCloud DNS RR shape used by cloudlist.
type DomainResourceRecord struct {
	ID              int    `json:"id"`
	HostRecord      string `json:"hostRecord"`
	HostValue       string `json:"hostValue"`
	Type            string `json:"type"`
	TTL             int    `json:"ttl"`
	MxPriority      int    `json:"mxPriority,omitempty"`
	Weight          int    `json:"weight,omitempty"`
	ViewValue       []int  `json:"viewValue,omitempty"`
	ResolvingStatus string `json:"resolvingStatus,omitempty"`
}

type DescribeResourceRecordResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		DataList     []DomainResourceRecord `json:"dataList"`
		CurrentCount int                    `json:"currentCount"`
		TotalCount   int                    `json:"totalCount"`
		TotalPage    int                    `json:"totalPage"`
	} `json:"result"`
}

// DescribeResourceRecord lists RRs under a domain. domainID is the integer ID
// returned by DescribeDomains, formatted as a string for the path.
func (c *Client) DescribeResourceRecord(ctx context.Context, region, domainID string, pageNumber, pageSize int) (DescribeResourceRecordResponse, error) {
	if region == "" || region == "all" {
		region = "cn-north-1"
	}
	query := url.Values{}
	if pageNumber > 0 {
		query.Set("pageNumber", strconv.Itoa(pageNumber))
	}
	if pageSize > 0 {
		query.Set("pageSize", strconv.Itoa(pageSize))
	}
	var resp DescribeResourceRecordResponse
	err := c.DoJSON(ctx, Request{
		Service:    "domainservice",
		Region:     region,
		Method:     http.MethodGet,
		Version:    "v2",
		Path:       "/regions/" + region + "/domain/" + domainID + "/ResourceRecord",
		Query:      query,
		Idempotent: true,
	}, &resp)
	return resp, err
}
