package s3

import (
	"context"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3Provider struct {
	Session *session.Session
}

func (d *S3Provider) GetBuckets(ctx context.Context) ([]*schema.Storage, error) {
	list := schema.NewResources().Storages
	select {
	case <-ctx.Done():
		return list, nil
	default:
		log.Println("[*] Start enumerating S3 ...")
	}
	client := s3.New(d.Session)
	buckets, err := client.ListBuckets(&s3.ListBucketsInput{})
	if err != nil {
		log.Println("[-] Enumerate S3 failed.")
		return list, err
	}
	for _, bucket := range buckets.Buckets {
		_bucket := &schema.Storage{BucketName: *bucket.Name}

		locationInput := &s3.GetBucketLocationInput{Bucket: bucket.Name}
		bucketLocation, err := client.GetBucketLocation(locationInput)
		if err != nil {
			log.Println("[-] Get bucket info failed.")
			return list, err
		}
		if bucketLocation.LocationConstraint != nil {
			_bucket.Region = *bucketLocation.LocationConstraint
		}
		list = append(list, _bucket)
		select {
		case <-ctx.Done():
			return list, nil
		}
	}

	return list, nil
}
