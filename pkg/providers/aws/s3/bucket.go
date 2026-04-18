package s3

import (
	"context"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Driver struct {
	Config awsv2.Config
}

func (d *Driver) GetBuckets(ctx context.Context) ([]schema.Storage, error) {
	list := []schema.Storage{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List S3 buckets ...")
	}
	client := s3.NewFromConfig(d.Config)
	buckets, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		logger.Error("List buckets failed.")
		return list, err
	}
	for _, bucket := range buckets.Buckets {
		_bucket := schema.Storage{BucketName: awsv2.ToString(bucket.Name)}

		locationInput := &s3.GetBucketLocationInput{Bucket: bucket.Name}
		bucketLocation, err := client.GetBucketLocation(ctx, locationInput)
		if err != nil {
			logger.Error("Get bucket info failed.")
			return list, err
		}
		if bucketLocation.LocationConstraint != "" {
			_bucket.Region = string(bucketLocation.LocationConstraint)
		}
		if _bucket.Region == "" {
			_bucket.Region = awsv2.ToString(bucket.BucketRegion)
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
