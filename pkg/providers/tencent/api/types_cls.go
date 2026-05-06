package api

import "context"

// Tencent Cloud Log Service (CLS) DescribeLogsets — lists log "logsets",
// the project-level container that holds CLS topics. Each logset corresponds
// to one cloudlist `log` asset row.
const clsAPIVersion = "2020-10-16"

type DescribeLogsetsRequest struct {
	Filters []CLSFilter `json:"Filters,omitempty"`
	Offset  *uint64     `json:"Offset,omitempty"`
	Limit   *uint64     `json:"Limit,omitempty"`
}

type CLSFilter struct {
	Key    *string  `json:"Key,omitempty"`
	Values []string `json:"Values,omitempty"`
}

type DescribeLogsetsResponse struct {
	Response struct {
		TotalCount *uint64     `json:"TotalCount"`
		Logsets    []CLSLogset `json:"Logsets"`
		RequestID  string      `json:"RequestId"`
	} `json:"Response"`
}

// CLSLogset is the typed wire shape of a CLS logset entry. The CLS API uses
// pointer fields for every value; helper getters live in the driver to
// dereference them safely.
type CLSLogset struct {
	LogsetID     *string `json:"LogsetId"`
	LogsetName   *string `json:"LogsetName"`
	CreateTime   *string `json:"CreateTime"`
	AssumerName  *string `json:"AssumerName"`
	Tags         []CLSTag `json:"Tags"`
	TopicCount   *uint64 `json:"TopicCount"`
	RoleName     *string `json:"RoleName"`
}

type CLSTag struct {
	Key   *string `json:"Key"`
	Value *string `json:"Value"`
}

// DescribeLogsets queries the CLS logsets in a region. limit/offset paginate.
func (c *Client) DescribeLogsets(ctx context.Context, region string, offset, limit uint64) (DescribeLogsetsResponse, error) {
	req := DescribeLogsetsRequest{}
	if offset > 0 {
		v := offset
		req.Offset = &v
	}
	if limit > 0 {
		v := limit
		req.Limit = &v
	}
	var resp DescribeLogsetsResponse
	err := c.DoJSON(ctx, "cls", clsAPIVersion, "DescribeLogsets", region, req, &resp)
	return resp, err
}
