// Package tls wraps the Volcengine TLS DescribeProjects endpoint for the
// cloudlist `log` asset.
package tls

import (
	"context"
	"errors"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const (
	defaultRegion = "cn-beijing"
	pageSize      = 100
	maxPages      = 50
)

// Driver lists TLS projects via the per-region Volcengine OpenAPI.
type Driver struct {
	Client *api.Client
	Region string
}

func (d *Driver) requestRegion() string {
	if r := d.Region; r != "" && r != "all" {
		return r
	}
	return defaultRegion
}

// GetLogs lists TLS projects in the configured region.
func (d *Driver) GetLogs(ctx context.Context) ([]schema.Log, error) {
	out := []schema.Log{}
	if d == nil || d.Client == nil {
		return out, errors.New("volcengine tls: nil api client")
	}
	logger.Info("List Volcengine TLS projects ...")
	region := d.requestRegion()
	for page := 1; page <= maxPages; page++ {
		resp, err := d.Client.DescribeTLSProjects(ctx, region, page, pageSize)
		if err != nil {
			return out, err
		}
		projects := resp.ProjectItems()
		for _, p := range projects {
			out = append(out, schema.Log{
				ProjectName:    p.ProjectName,
				Region:         firstNonEmpty(p.Region, region),
				Description:    p.Description,
				LastModifyTime: p.CreateTime,
			})
		}
		if len(projects) < pageSize {
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
