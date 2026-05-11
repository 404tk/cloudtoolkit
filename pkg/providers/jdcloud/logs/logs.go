// Package logs wraps the JDCloud log service for the cloudlist `log` asset.
package logs

import (
	"context"
	"errors"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const (
	defaultRegion = "cn-north-1"
	pageSize      = 100
	maxPages      = 50
)

type Driver struct {
	Client *api.Client
	Region string
}

func (d *Driver) requestRegion() string {
	if r := strings.TrimSpace(d.Region); r != "" && r != "all" {
		return r
	}
	return defaultRegion
}

// GetLogs lists JDCloud logsets in the resolved region.
func (d *Driver) GetLogs(ctx context.Context) ([]schema.Log, error) {
	out := []schema.Log{}
	if d == nil || d.Client == nil {
		return out, errors.New("jdcloud logs: nil api client")
	}
	logger.Info("List JDCloud logsets ...")
	region := d.requestRegion()
	for page := 1; page <= maxPages; page++ {
		resp, err := d.Client.DescribeLogsets(ctx, region, page, pageSize)
		if err != nil {
			return out, err
		}
		for _, s := range resp.Result.Data {
			out = append(out, schema.Log{
				ProjectName:    s.Name,
				Region:         firstNonEmpty(s.Region, region),
				Description:    firstNonEmpty(s.Description, s.UID),
				LastModifyTime: s.CreateTime,
			})
		}
		if len(resp.Result.Data) < pageSize {
			break
		}
	}
	return out, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
