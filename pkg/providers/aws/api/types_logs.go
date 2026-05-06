package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// AWS CloudWatch Logs DescribeLogGroups — JSON-1.1 RPC like CloudTrail.
//
//	X-Amz-Target: Logs_20140328.DescribeLogGroups
//	Endpoint:     logs.<region>.amazonaws.com
const (
	cloudWatchLogsContentType  = "application/x-amz-json-1.1"
	cloudWatchLogsDescribeLogG = "Logs_20140328.DescribeLogGroups"
)

type DescribeLogGroupsInput struct {
	LogGroupNamePrefix *string `json:"logGroupNamePrefix,omitempty"`
	NextToken          *string `json:"nextToken,omitempty"`
	Limit              *int64  `json:"limit,omitempty"`
}

type LogGroup struct {
	LogGroupName    string `json:"logGroupName"`
	CreationTime    int64  `json:"creationTime"`
	RetentionInDays int64  `json:"retentionInDays"`
	StoredBytes     int64  `json:"storedBytes"`
	Arn             string `json:"arn"`
}

// CreationTimeFormatted converts CloudWatch's millisecond epoch into the
// `YYYY-MM-DD HH:MM:SS` format the cloudlist `log` asset expects.
func (g LogGroup) CreationTimeFormatted() string {
	if g.CreationTime <= 0 {
		return ""
	}
	t := time.Unix(g.CreationTime/1000, 0).UTC()
	return t.Format("2006-01-02 15:04:05")
}

type DescribeLogGroupsOutput struct {
	LogGroups []LogGroup `json:"logGroups"`
	NextToken string     `json:"nextToken"`
}

// CloudWatchLogsDescribeLogGroups lists log groups in `region`. nextToken
// paginates; pass "" for the first call.
func (c *Client) CloudWatchLogsDescribeLogGroups(ctx context.Context, region string, limit int64, nextToken string) (DescribeLogGroupsOutput, error) {
	input := DescribeLogGroupsInput{}
	if limit > 0 {
		v := limit
		input.Limit = &v
	}
	if nextToken != "" {
		t := nextToken
		input.NextToken = &t
	}
	body, err := json.Marshal(input)
	if err != nil {
		return DescribeLogGroupsOutput{}, err
	}
	headers := http.Header{}
	headers.Set("Content-Type", cloudWatchLogsContentType)
	headers.Set("X-Amz-Target", cloudWatchLogsDescribeLogG)
	var out DescribeLogGroupsOutput
	err = c.DoRESTJSON(ctx, Request{
		Service:    "logs",
		Region:     region,
		Method:     http.MethodPost,
		Path:       "/",
		Body:       body,
		Headers:    headers,
		Idempotent: true,
	}, &out)
	return out, err
}
