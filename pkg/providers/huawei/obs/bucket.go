package obs

import (
	"context"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
)

type Driver struct {
	Auth    basic.Credentials
	Regions []string
}

func (d *Driver) GetBuckets(ctx context.Context) ([]schema.Storage, error) {
	list := []schema.Storage{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List OBS buckets...")
	}
	for _, r := range d.Regions {
		endPoint := "obs." + r + ".myhuaweicloud.com"
		client, err := obs.New(d.Auth.AK, d.Auth.SK, endPoint)
		if err != nil {
			continue
		}
		response, err := client.ListBuckets(nil)
		if err != nil {
			logger.Error("List buckets failed with", r)
			return list, err
		}

		for _, bucket := range response.Buckets {
			_bucket := schema.Storage{
				BucketName: bucket.Name,
				Region:     bucket.Location,
			}
			if _bucket.Region == "" {
				_bucket.Region = r
			}
			list = append(list, _bucket)
		}
		if len(list) > 0 {
			break
		}
	}

	return list, nil
}
