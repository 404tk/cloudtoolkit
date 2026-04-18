package api

import "context"

const (
	cvmVersion             = "2017-03-12"
	defaultCVMListLimit    = 100
	defaultCVMRegionTarget = DefaultRegion
)

type DescribeCVMRegionsRequest struct{}

type DescribeCVMRegionsResponse struct {
	Response struct {
		RegionSet []CVMRegionInfo `json:"RegionSet"`
		RequestID string          `json:"RequestId"`
	} `json:"Response"`
}

type CVMRegionInfo struct {
	Region *string `json:"Region"`
}

func (c *Client) DescribeCVMRegions(ctx context.Context, region string) (DescribeCVMRegionsResponse, error) {
	if region == "" || region == "all" {
		region = defaultCVMRegionTarget
	}
	var resp DescribeCVMRegionsResponse
	err := c.DoJSON(ctx, "cvm", cvmVersion, "DescribeRegions", region, DescribeCVMRegionsRequest{}, &resp)
	return resp, err
}

type DescribeCVMInstancesRequest struct {
	Offset *int64 `json:"Offset,omitempty"`
	Limit  *int64 `json:"Limit,omitempty"`
}

type DescribeCVMInstancesResponse struct {
	Response struct {
		TotalCount  *int64            `json:"TotalCount"`
		InstanceSet []CVMInstanceInfo `json:"InstanceSet"`
		RequestID   string            `json:"RequestId"`
	} `json:"Response"`
}

type CVMInstanceInfo struct {
	InstanceID         *string  `json:"InstanceId"`
	InstanceName       *string  `json:"InstanceName"`
	InstanceState      *string  `json:"InstanceState"`
	PublicIPAddresses  []string `json:"PublicIpAddresses"`
	PrivateIPAddresses []string `json:"PrivateIpAddresses"`
	OSName             *string  `json:"OsName"`
}

func (c *Client) DescribeCVMInstances(ctx context.Context, region string, offset, limit int64) (DescribeCVMInstancesResponse, error) {
	if limit <= 0 {
		limit = defaultCVMListLimit
	}
	var resp DescribeCVMInstancesResponse
	err := c.DoJSON(
		ctx,
		"cvm",
		cvmVersion,
		"DescribeInstances",
		normalizeRegion(region),
		DescribeCVMInstancesRequest{
			Offset: int64Ptr(offset),
			Limit:  int64Ptr(limit),
		},
		&resp,
	)
	return resp, err
}
