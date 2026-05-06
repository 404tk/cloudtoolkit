package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// Volcengine TLS (Tencent-style "Topic Log Service") project listing —
// roughly equivalent to AWS CloudWatch Logs log groups or Tencent CLS
// logsets. The cloudlist `log` asset carries one row per project.
const tlsAPIVersion = "2020-01-01"

// TLSProject is a single TLS project record.
type TLSProject struct {
	ProjectID    string `json:"ProjectId"`
	ProjectName  string `json:"ProjectName"`
	Region       string `json:"Region"`
	CreateTime   string `json:"CreateTime"`
	Description  string `json:"Description"`
	IamProjectID string `json:"IamProjectName"`
}

type DescribeTLSProjectsResponse struct {
	ResponseMetadata ResponseMetadata `json:"ResponseMetadata"`
	Result           struct {
		Total    int          `json:"Total"`
		Projects []TLSProject `json:"Projects"`
	} `json:"Result"`
}

// DescribeTLSProjects lists TLS projects in a region. Volcengine's TLS API is
// per-region; cloudlist callers iterate across regions externally.
func (c *Client) DescribeTLSProjects(ctx context.Context, region string, pageNumber, pageSize int) (DescribeTLSProjectsResponse, error) {
	query := url.Values{}
	if pageNumber > 0 {
		query.Set("PageNumber", strconv.Itoa(pageNumber))
	}
	if pageSize > 0 {
		query.Set("PageSize", strconv.Itoa(pageSize))
	}
	var out DescribeTLSProjectsResponse
	err := c.DoOpenAPI(ctx, Request{
		Service:    "tls",
		Version:    tlsAPIVersion,
		Action:     "DescribeProjects",
		Method:     http.MethodGet,
		Region:     region,
		Path:       "/",
		Query:      query,
		Idempotent: true,
	}, &out)
	return out, err
}
