// Package cls wraps Tencent Cloud Log Service for the cloudlist `log`
// asset. CLS organises logs hierarchically: logset → topic. Listing logsets
// is the cheapest project-level summary that maps cleanly to the schema.Log
// shape used by other providers.
package cls

import (
	"context"
	"errors"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const (
	defaultRegion = "ap-guangzhou"
	pageSize      = 100
	maxPages      = 50
)

type Driver struct {
	Credential    auth.Credential
	Region        string
	clientOptions []api.Option
}

func (d *Driver) SetClientOptions(opts ...api.Option) {
	d.clientOptions = append([]api.Option(nil), opts...)
}

func (d *Driver) newClient() *api.Client {
	return api.NewClient(d.Credential, d.clientOptions...)
}

func (d *Driver) requestRegion() string {
	if d == nil {
		return defaultRegion
	}
	if r := d.Region; r != "" && r != "all" {
		return r
	}
	return defaultRegion
}

// GetLogs lists CLS logsets and surfaces them as cloudlist `log` rows.
func (d *Driver) GetLogs(ctx context.Context) ([]schema.Log, error) {
	out := []schema.Log{}
	if d == nil {
		return out, errors.New("tencent cls: nil driver")
	}
	logger.Info("List Tencent CLS logsets ...")
	region := d.requestRegion()
	client := d.newClient()
	offset := uint64(0)
	for page := 0; page < maxPages; page++ {
		resp, err := client.DescribeLogsets(ctx, region, offset, pageSize)
		if err != nil {
			return out, err
		}
		for _, ls := range resp.Response.Logsets {
			out = append(out, schema.Log{
				ProjectName:    derefString(ls.LogsetName),
				Region:         region,
				Description:    describeLogset(ls),
				LastModifyTime: derefString(ls.CreateTime),
			})
		}
		if uint64(len(resp.Response.Logsets)) < pageSize {
			break
		}
		offset += uint64(len(resp.Response.Logsets))
	}
	return out, nil
}

func describeLogset(ls api.CLSLogset) string {
	id := derefString(ls.LogsetID)
	if id == "" {
		return ""
	}
	return "logsetId=" + id
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
