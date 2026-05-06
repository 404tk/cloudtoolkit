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

type CDBAccount struct {
	User       *string `json:"User"`
	Host       *string `json:"Host"`
	Notes      *string `json:"Notes,omitempty"`
	CreateTime *string `json:"CreateTime,omitempty"`
}

type DescribeCDBAccountsRequest struct {
	InstanceID *string `json:"InstanceId,omitempty"`
	Limit      *int64  `json:"Limit,omitempty"`
	Offset     *int64  `json:"Offset,omitempty"`
}

type DescribeCDBAccountsResponse struct {
	Response struct {
		TotalCount *int64       `json:"TotalCount"`
		Items      []CDBAccount `json:"Items"`
		RequestID  string       `json:"RequestId"`
	} `json:"Response"`
}

type CreateCDBAccountsRequest struct {
	InstanceID *string      `json:"InstanceId,omitempty"`
	Accounts   []CDBAccount `json:"Accounts,omitempty"`
	Password   *string      `json:"Password,omitempty"`
	Description *string     `json:"Description,omitempty"`
}

type CreateCDBAccountsResponse struct {
	Response struct {
		AsyncRequestID *string `json:"AsyncRequestId"`
		RequestID      string  `json:"RequestId"`
	} `json:"Response"`
}

type DeleteCDBAccountsRequest struct {
	InstanceID *string      `json:"InstanceId,omitempty"`
	Accounts   []CDBAccount `json:"Accounts,omitempty"`
}

type DeleteCDBAccountsResponse struct {
	Response struct {
		AsyncRequestID *string `json:"AsyncRequestId"`
		RequestID      string  `json:"RequestId"`
	} `json:"Response"`
}

func (c *Client) DescribeCDBAccounts(ctx context.Context, region, instanceID string) (DescribeCDBAccountsResponse, error) {
	limit := int64(20)
	offset := int64(0)
	req := DescribeCDBAccountsRequest{
		InstanceID: stringPtr(instanceID),
		Limit:      &limit,
		Offset:     &offset,
	}
	var resp DescribeCDBAccountsResponse
	err := c.DoJSON(ctx, "cdb", cdbVersion, "DescribeAccounts", normalizeRegion(region), req, &resp)
	return resp, err
}

func (c *Client) CreateCDBAccounts(ctx context.Context, region, instanceID, user, host, password string) (CreateCDBAccountsResponse, error) {
	req := CreateCDBAccountsRequest{
		InstanceID: stringPtr(instanceID),
		Password:   stringPtr(password),
		Accounts: []CDBAccount{{
			User: stringPtr(user),
			Host: stringPtr(host),
		}},
	}
	var resp CreateCDBAccountsResponse
	err := c.DoJSON(ctx, "cdb", cdbVersion, "CreateAccounts", normalizeRegion(region), req, &resp)
	return resp, err
}

func (c *Client) DeleteCDBAccounts(ctx context.Context, region, instanceID, user, host string) (DeleteCDBAccountsResponse, error) {
	req := DeleteCDBAccountsRequest{
		InstanceID: stringPtr(instanceID),
		Accounts: []CDBAccount{{
			User: stringPtr(user),
			Host: stringPtr(host),
		}},
	}
	var resp DeleteCDBAccountsResponse
	err := c.DoJSON(ctx, "cdb", cdbVersion, "DeleteAccounts", normalizeRegion(region), req, &resp)
	return resp, err
}
