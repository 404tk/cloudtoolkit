package oss

import (
	"context"
	"errors"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Client *api.Client
}

func (d *Driver) ListBuckets(ctx context.Context) ([]schema.Storage, error) {
	list := []schema.Storage{}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List OSS buckets ...")
	}
	if d.Client == nil {
		return list, errors.New("jdcloud oss: nil api client")
	}

	var resp api.ListBucketsResponse
	err := d.Client.DoJSON(ctx, api.Request{
		Service: "oss",
		Region:  "cn-north-1",
		Method:  "GET",
		Version: "v1",
		Path:    "/regions/cn-north-1/buckets",
	}, &resp)
	if err != nil {
		logger.Error("List buckets failed.")
		return list, err
	}

	for _, bucket := range resp.Result.Buckets {
		_bucket := schema.Storage{
			BucketName: bucket.Name,
		}
		list = append(list, _bucket)
	}

	return list, nil
}
