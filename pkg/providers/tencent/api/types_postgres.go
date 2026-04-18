package api

import "context"

const postgresVersion = "2017-03-12"

type DescribePostgresRegionsRequest struct{}

type DescribePostgresRegionsResponse struct {
	Response struct {
		RegionSet []PostgresRegionInfo `json:"RegionSet"`
		RequestID string               `json:"RequestId"`
	} `json:"Response"`
}

type PostgresRegionInfo struct {
	Region      *string `json:"Region"`
	RegionState *string `json:"RegionState"`
}

func (c *Client) DescribePostgresRegions(ctx context.Context, region string) (DescribePostgresRegionsResponse, error) {
	var resp DescribePostgresRegionsResponse
	err := c.DoJSON(
		ctx,
		"postgres",
		postgresVersion,
		"DescribeRegions",
		normalizeRegion(region),
		DescribePostgresRegionsRequest{},
		&resp,
	)
	return resp, err
}

type DescribePostgresInstancesRequest struct{}

type DescribePostgresInstancesResponse struct {
	Response struct {
		DBInstanceSet []PostgresInstanceInfo `json:"DBInstanceSet"`
		RequestID     string                 `json:"RequestId"`
	} `json:"Response"`
}

type PostgresInstanceInfo struct {
	DBInstanceID      *string           `json:"DBInstanceId"`
	DBEngine          *string           `json:"DBEngine"`
	DBInstanceVersion *string           `json:"DBInstanceVersion"`
	Region            *string           `json:"Region"`
	DBInstanceNetInfo []PostgresNetInfo `json:"DBInstanceNetInfo"`
}

type PostgresNetInfo struct {
	Address *string `json:"Address"`
	IP      *string `json:"Ip"`
	Port    *uint64 `json:"Port"`
	NetType *string `json:"NetType"`
	Status  *string `json:"Status"`
}

func (c *Client) DescribePostgresInstances(ctx context.Context, region string) (DescribePostgresInstancesResponse, error) {
	var resp DescribePostgresInstancesResponse
	err := c.DoJSON(
		ctx,
		"postgres",
		postgresVersion,
		"DescribeDBInstances",
		normalizeRegion(region),
		DescribePostgresInstancesRequest{},
		&resp,
	)
	return resp, err
}
