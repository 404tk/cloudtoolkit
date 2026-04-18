package api

import "context"

const sqlserverVersion = "2018-03-28"

type DescribeSQLServerRegionsRequest struct{}

type DescribeSQLServerRegionsResponse struct {
	Response struct {
		RegionSet []SQLServerRegionInfo `json:"RegionSet"`
		RequestID string                `json:"RequestId"`
	} `json:"Response"`
}

type SQLServerRegionInfo struct {
	Region      *string `json:"Region"`
	RegionState *string `json:"RegionState"`
}

func (c *Client) DescribeSQLServerRegions(ctx context.Context, region string) (DescribeSQLServerRegionsResponse, error) {
	var resp DescribeSQLServerRegionsResponse
	err := c.DoJSON(
		ctx,
		"sqlserver",
		sqlserverVersion,
		"DescribeRegions",
		normalizeRegion(region),
		DescribeSQLServerRegionsRequest{},
		&resp,
	)
	return resp, err
}

type DescribeSQLServerInstancesRequest struct{}

type DescribeSQLServerInstancesResponse struct {
	Response struct {
		DBInstances []SQLServerInstanceInfo `json:"DBInstances"`
		RequestID   string                  `json:"RequestId"`
	} `json:"Response"`
}

type SQLServerInstanceInfo struct {
	InstanceID   *string `json:"InstanceId"`
	VersionName  *string `json:"VersionName"`
	Version      *string `json:"Version"`
	Region       *string `json:"Region"`
	DNSPodDomain *string `json:"DnsPodDomain"`
	TgwWanVPort  *int64  `json:"TgwWanVPort"`
	Vip          *string `json:"Vip"`
	Vport        *int64  `json:"Vport"`
}

func (c *Client) DescribeSQLServerInstances(ctx context.Context, region string) (DescribeSQLServerInstancesResponse, error) {
	var resp DescribeSQLServerInstancesResponse
	err := c.DoJSON(
		ctx,
		"sqlserver",
		sqlserverVersion,
		"DescribeDBInstances",
		normalizeRegion(region),
		DescribeSQLServerInstancesRequest{},
		&resp,
	)
	return resp, err
}
