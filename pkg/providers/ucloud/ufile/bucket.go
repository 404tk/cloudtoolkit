package ufile

import (
	"context"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	ucloudauth "github.com/404tk/cloudtoolkit/pkg/providers/ucloud/auth"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const pageSize = 100

type Driver struct {
	Credential ucloudauth.Credential
	Client     *api.Client
	ProjectID  string
	Region     string
}

func (d *Driver) GetBuckets(ctx context.Context) ([]schema.Storage, error) {
	list := []schema.Storage{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List UFile buckets ...")
	}

	return paginate.Fetch[schema.Storage, int](ctx, func(ctx context.Context, offset int) (paginate.Page[schema.Storage, int], error) {
		params := map[string]any{
			"Limit":  pageSize,
			"Offset": offset,
		}
		if region := strings.TrimSpace(d.Region); region != "" && !strings.EqualFold(region, "all") {
			params["Region"] = region
		}

		var resp api.DescribeBucketResponse
		err := d.client().Do(ctx, api.Request{
			Action: "DescribeBucket",
			Params: params,
		}, &resp)
		if err != nil {
			return paginate.Page[schema.Storage, int]{}, err
		}

		items := make([]schema.Storage, 0, len(resp.DataSet))
		for _, bucket := range resp.DataSet {
			region := strings.TrimSpace(bucket.Region)
			if region == "" && strings.TrimSpace(d.Region) != "" && !strings.EqualFold(strings.TrimSpace(d.Region), "all") {
				region = strings.TrimSpace(d.Region)
			}
			items = append(items, schema.Storage{
				BucketName: strings.TrimSpace(bucket.BucketName),
				Region:     region,
			})
		}

		next := offset + len(items)
		done := len(items) == 0 || len(items) < pageSize
		return paginate.Page[schema.Storage, int]{
			Items: items,
			Next:  next,
			Done:  done,
		}, nil
	})
}

func (d *Driver) client() *api.Client {
	if d.Client != nil {
		return d.Client
	}
	return api.NewClient(d.Credential, api.WithProjectID(d.ProjectID))
}
