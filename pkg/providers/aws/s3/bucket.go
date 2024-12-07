package s3

import (
	"context"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type Driver struct {
	Session *session.Session
}

func (d *Driver) GetBuckets(ctx context.Context) ([]schema.Storage, error) {
	list := []schema.Storage{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List S3 buckets ...")
	}
	client := s3.New(d.Session)
	buckets, err := client.ListBuckets(&s3.ListBucketsInput{})
	if err != nil {
		logger.Error("List buckets failed.")
		return list, err
	}
	for _, bucket := range buckets.Buckets {
		_bucket := schema.Storage{BucketName: *bucket.Name}

		locationInput := &s3.GetBucketLocationInput{Bucket: bucket.Name}
		bucketLocation, err := client.GetBucketLocation(locationInput)
		if err != nil {
			logger.Error("Get bucket info failed.")
			return list, err
		}
		if bucketLocation.LocationConstraint != nil {
			_bucket.Region = *bucketLocation.LocationConstraint
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
