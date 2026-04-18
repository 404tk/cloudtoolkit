package s3

import (
	"context"
	"errors"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Client        *api.Client
	DefaultRegion string
}

var errNilAPIClient = errors.New("aws s3: nil api client")

func (d *Driver) GetBuckets(ctx context.Context) ([]schema.Storage, error) {
	list := []schema.Storage{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List S3 buckets ...")
	}
	client, err := d.requireClient()
	if err != nil {
		return list, err
	}
	buckets, err := client.ListBuckets(ctx, d.defaultRegion())
	if err != nil {
		logger.Error("List buckets failed.")
		return list, err
	}
	for _, bucket := range buckets.Buckets {
		_bucket := schema.Storage{BucketName: bucket.Name}
		bucketLocation, err := client.GetBucketLocation(ctx, d.defaultRegion(), bucket.Name)
		if err != nil {
			logger.Error("Get bucket info failed.")
			return list, err
		}
		if bucketLocation.Region != "" {
			_bucket.Region = bucketLocation.Region
		}
		if _bucket.Region == "" {
			_bucket.Region = bucket.BucketRegion
		}
		if _bucket.Region == "" {
			_bucket.Region = d.defaultRegion()
		}
		list = append(list, _bucket)
		select {
		case <-ctx.Done():
			return list, nil
		default:
			continue
		}
	}

	return list, nil
}

func (d *Driver) requireClient() (*api.Client, error) {
	if d.Client == nil {
		return nil, errNilAPIClient
	}
	return d.Client, nil
}

func (d *Driver) defaultRegion() string {
	if d.DefaultRegion == "" {
		return "us-east-1"
	}
	return d.DefaultRegion
}
