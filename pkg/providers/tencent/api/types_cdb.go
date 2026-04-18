package api

import "context"

const cdbVersion = "2017-03-20"

type DescribeCDBZoneConfigRequest struct{}

type DescribeCDBZoneConfigResponse struct {
	Response struct {
		DataResult struct {
			Regions []CDBRegionSellConf `json:"Regions"`
		} `json:"DataResult"`
		RequestID string `json:"RequestId"`
	} `json:"Response"`
}

type CDBRegionSellConf struct {
	Region *string `json:"Region"`
}

func (c *Client) DescribeCDBZoneConfig(ctx context.Context, region string) (DescribeCDBZoneConfigResponse, error) {
	var resp DescribeCDBZoneConfigResponse
	err := c.DoJSON(
		ctx,
		"cdb",
		cdbVersion,
		"DescribeCdbZoneConfig",
		normalizeRegion(region),
		DescribeCDBZoneConfigRequest{},
		&resp,
	)
	return resp, err
}

type DescribeCDBInstancesRequest struct{}

type DescribeCDBInstancesResponse struct {
	Response struct {
		Items     []CDBInstanceInfo `json:"Items"`
		RequestID string            `json:"RequestId"`
	} `json:"Response"`
}

type CDBInstanceInfo struct {
	InstanceID    *string `json:"InstanceId"`
	EngineVersion *string `json:"EngineVersion"`
	Region        *string `json:"Region"`
	WanStatus     *int64  `json:"WanStatus"`
	WanDomain     *string `json:"WanDomain"`
	WanPort       *int64  `json:"WanPort"`
	Vip           *string `json:"Vip"`
	Vport         *int64  `json:"Vport"`
}

func (c *Client) DescribeCDBInstances(ctx context.Context, region string) (DescribeCDBInstancesResponse, error) {
	var resp DescribeCDBInstancesResponse
	err := c.DoJSON(
		ctx,
		"cdb",
		cdbVersion,
		"DescribeDBInstances",
		normalizeRegion(region),
		DescribeCDBInstancesRequest{},
		&resp,
	)
	return resp, err
}
