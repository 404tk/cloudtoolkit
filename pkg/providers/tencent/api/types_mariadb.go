package api

import "context"

const mariadbVersion = "2017-03-12"

type DescribeMariaDBSaleInfoRequest struct{}

type DescribeMariaDBSaleInfoResponse struct {
	Response struct {
		RegionList []MariaDBRegionInfo `json:"RegionList"`
		RequestID  string              `json:"RequestId"`
	} `json:"Response"`
}

type MariaDBRegionInfo struct {
	Region *string `json:"Region"`
}

func (c *Client) DescribeMariaDBSaleInfo(ctx context.Context, region string) (DescribeMariaDBSaleInfoResponse, error) {
	var resp DescribeMariaDBSaleInfoResponse
	err := c.DoJSON(
		ctx,
		"mariadb",
		mariadbVersion,
		"DescribeSaleInfo",
		normalizeRegion(region),
		DescribeMariaDBSaleInfoRequest{},
		&resp,
	)
	return resp, err
}

type DescribeMariaDBInstancesRequest struct{}

type DescribeMariaDBInstancesResponse struct {
	Response struct {
		Instances []MariaDBInstanceInfo `json:"Instances"`
		RequestID string                `json:"RequestId"`
	} `json:"Response"`
}

type MariaDBInstanceInfo struct {
	InstanceID *string `json:"InstanceId"`
	DBVersion  *string `json:"DbVersion"`
	Region     *string `json:"Region"`
	WanStatus  *int64  `json:"WanStatus"`
	WanDomain  *string `json:"WanDomain"`
	WanPort    *int64  `json:"WanPort"`
	Vip        *string `json:"Vip"`
	Vport      *int64  `json:"Vport"`
}

func (c *Client) DescribeMariaDBInstances(ctx context.Context, region string) (DescribeMariaDBInstancesResponse, error) {
	var resp DescribeMariaDBInstancesResponse
	err := c.DoJSON(
		ctx,
		"mariadb",
		mariadbVersion,
		"DescribeDBInstances",
		normalizeRegion(region),
		DescribeMariaDBInstancesRequest{},
		&resp,
	)
	return resp, err
}
