package api

import (
	"context"
	"net/http"
)

const billingAPIVersion = "2022-01-01"

type QueryBalanceAcctResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		AvailableBalance string `json:"AvailableBalance"`
	} `json:"Result"`
}

func (c *Client) QueryBalanceAcct(ctx context.Context, region string) (QueryBalanceAcctResponse, error) {
	var out QueryBalanceAcctResponse
	err := c.DoOpenAPI(ctx, Request{
		Service:    "billing",
		Version:    billingAPIVersion,
		Action:     "QueryBalanceAcct",
		Method:     http.MethodPost,
		Region:     region,
		Path:       "/",
		Body:       []byte("{}"),
		Idempotent: true,
	}, &out)
	return out, err
}
