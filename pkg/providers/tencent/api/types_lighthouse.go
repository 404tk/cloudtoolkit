package api

import "context"

const (
	lighthouseVersion             = "2020-03-24"
	defaultLighthouseListLimit    = 100
	defaultLighthouseRegionTarget = DefaultRegion
)

type DescribeLighthouseRegionsRequest struct{}

type DescribeLighthouseRegionsResponse struct {
	Response struct {
		RegionSet []LighthouseRegionInfo `json:"RegionSet"`
		RequestID string                 `json:"RequestId"`
	} `json:"Response"`
}

type LighthouseRegionInfo struct {
	Region *string `json:"Region"`
}

func (c *Client) DescribeLighthouseRegions(ctx context.Context, region string) (DescribeLighthouseRegionsResponse, error) {
	if region == "" || region == "all" {
		region = defaultLighthouseRegionTarget
	}
	var resp DescribeLighthouseRegionsResponse
	err := c.DoJSON(
		ctx,
		"lighthouse",
		lighthouseVersion,
		"DescribeRegions",
		region,
		DescribeLighthouseRegionsRequest{},
		&resp,
	)
	return resp, err
}

type DescribeLighthouseInstancesRequest struct {
	Offset *int64 `json:"Offset,omitempty"`
	Limit  *int64 `json:"Limit,omitempty"`
}

type DescribeLighthouseInstancesResponse struct {
	Response struct {
		TotalCount  *int64                   `json:"TotalCount"`
		InstanceSet []LighthouseInstanceInfo `json:"InstanceSet"`
		RequestID   string                   `json:"RequestId"`
	} `json:"Response"`
}

type LighthouseInstanceInfo struct {
	InstanceID       *string  `json:"InstanceId"`
	InstanceName     *string  `json:"InstanceName"`
	InstanceState    *string  `json:"InstanceState"`
	PublicAddresses  []string `json:"PublicAddresses"`
	PrivateAddresses []string `json:"PrivateAddresses"`
	PlatformType     *string  `json:"PlatformType"`
}

func (c *Client) DescribeLighthouseInstances(ctx context.Context, region string, offset, limit int64) (DescribeLighthouseInstancesResponse, error) {
	if limit <= 0 {
		limit = defaultLighthouseListLimit
	}
	var resp DescribeLighthouseInstancesResponse
	err := c.DoJSON(
		ctx,
		"lighthouse",
		lighthouseVersion,
		"DescribeInstances",
		normalizeRegion(region),
		DescribeLighthouseInstancesRequest{
			Offset: int64Ptr(offset),
			Limit:  int64Ptr(limit),
		},
		&resp,
	)
	return resp, err
}
