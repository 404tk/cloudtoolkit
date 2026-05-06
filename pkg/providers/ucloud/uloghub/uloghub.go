// Package uloghub wraps UCloud ULogHub DescribeULogTopic for the cloudlist
// `log` asset. Pattern-inferred — see api/types_uloghub.go.
package uloghub

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	ucloudauth "github.com/404tk/cloudtoolkit/pkg/providers/ucloud/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const (
	defaultRegion = "cn-bj2"
	pageSize      = 100
	maxPages      = 50
)

type Driver struct {
	Credential ucloudauth.Credential
	Client     *api.Client
	ProjectID  string
	Region     string
}

func (d *Driver) client() *api.Client {
	if d.Client != nil {
		return d.Client
	}
	return api.NewClient(d.Credential)
}

func (d *Driver) requestRegion() string {
	if r := strings.TrimSpace(d.Region); r != "" && r != "all" {
		return r
	}
	return defaultRegion
}

// GetLogs lists ULogHub log topics in the configured region.
func (d *Driver) GetLogs(ctx context.Context) ([]schema.Log, error) {
	out := []schema.Log{}
	if d == nil {
		return out, errors.New("ucloud uloghub: nil driver")
	}
	logger.Info("List UCloud ULogHub topics ...")
	region := d.requestRegion()
	offset := 0
	for page := 0; page < maxPages; page++ {
		params := map[string]any{
			"Region": region,
			"Limit":  strconv.Itoa(pageSize),
			"Offset": strconv.Itoa(offset),
		}
		if d.ProjectID != "" {
			params["ProjectId"] = d.ProjectID
		}
		var resp api.DescribeULogTopicResponse
		err := d.client().Do(ctx, api.Request{
			Action:     "DescribeULogTopic",
			Params:     params,
			Idempotent: true,
		}, &resp)
		if err != nil {
			return out, err
		}
		for _, t := range resp.Topics {
			name := t.TopicName
			if t.LogSetName != "" {
				name = t.LogSetName + "/" + t.TopicName
			}
			out = append(out, schema.Log{
				ProjectName:    name,
				Region:         firstNonEmpty(t.Region, region),
				Description:    t.TopicID,
				LastModifyTime: formatEpoch(firstPositive(t.UpdateTime, t.CreateTime)),
			})
		}
		if len(resp.Topics) < pageSize {
			break
		}
		offset += len(resp.Topics)
	}
	return out, nil
}

func formatEpoch(epoch int64) string {
	if epoch <= 0 {
		return ""
	}
	// UCloud surfaces both seconds and milliseconds depending on service age;
	// normalise large values down to seconds before formatting.
	if epoch > 1_000_000_000_000 {
		epoch /= 1000
	}
	return time.Unix(epoch, 0).UTC().Format("2006-01-02 15:04:05")
}

func firstPositive(values ...int64) int64 {
	for _, v := range values {
		if v > 0 {
			return v
		}
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
