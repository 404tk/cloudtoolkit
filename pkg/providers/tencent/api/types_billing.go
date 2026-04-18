package api

import "context"

const DefaultRegion = "ap-guangzhou"

type DescribeAccountBalanceRequest struct{}

type DescribeAccountBalanceResponse struct {
	Response struct {
		Balance            *int64   `json:"Balance"`
		RealBalance        *float64 `json:"RealBalance"`
		CashAccountBalance *float64 `json:"CashAccountBalance"`
		RequestID          string   `json:"RequestId"`
	} `json:"Response"`
}

func (c *Client) DescribeAccountBalance(ctx context.Context, region string) (DescribeAccountBalanceResponse, error) {
	var resp DescribeAccountBalanceResponse
	err := c.DoJSON(
		ctx,
		"billing",
		"2018-07-09",
		"DescribeAccountBalance",
		normalizeRegion(region),
		DescribeAccountBalanceRequest{},
		&resp,
	)
	return resp, err
}

func normalizeRegion(region string) string {
	switch region {
	case "", "all":
		return DefaultRegion
	default:
		return region
	}
}
