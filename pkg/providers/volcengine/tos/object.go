package tos

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

func (d *Driver) ListObjects(ctx context.Context, buckets map[string]string) ([]schema.BucketResult, error) {
	client, err := d.NewClient()
	if err != nil {
		return nil, err
	}

	results := make([]schema.BucketResult, 0, len(buckets))
	var errs []string
	for bucket, region := range buckets {
		token := ""
		objects := make([]schema.BucketObject, 0)
		failed := false
		for {
			resp, err := client.ListObjectsV2(ctx, bucket, normalizeBucketRegion(region), token, 100)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", bucket, err))
				failed = true
				break
			}

			for _, object := range resp.Contents {
				objects = append(objects, schema.BucketObject{
					BucketName:   bucket,
					Key:          object.Key,
					Size:         object.Size,
					LastModified: object.LastModified,
					StorageClass: object.StorageClass,
				})
			}

			if !resp.IsTruncated {
				break
			}
			token = strings.TrimSpace(resp.NextContinuationToken)
			if token == "" {
				errs = append(errs, fmt.Sprintf("%s: missing next continuation token", bucket))
				failed = true
				break
			}

			select {
			case <-ctx.Done():
				results = append(results, schema.BucketResult{
					Action:      "list",
					BucketName:  bucket,
					ObjectCount: int64(len(objects)),
					Objects:     objects,
					Message:     fmt.Sprintf("%d objects found", len(objects)),
				})
				return results, nil
			default:
			}
		}
		if !failed || len(objects) > 0 {
			results = append(results, schema.BucketResult{
				Action:      "list",
				BucketName:  bucket,
				ObjectCount: int64(len(objects)),
				Objects:     objects,
				Message:     fmt.Sprintf("%d objects found", len(objects)),
			})
		}
	}
	if len(errs) > 0 {
		return results, errors.New(strings.Join(errs, "; "))
	}
	return results, nil
}

func (d *Driver) TotalObjects(ctx context.Context, buckets map[string]string) ([]schema.BucketResult, error) {
	tracker := processbar.NewCountTracker()
	defer tracker.Finish()

	results := make([]schema.BucketResult, 0, len(buckets))
	var errs []string
	for bucket, region := range buckets {
		count, err := d.countBucketObjects(ctx, bucket, normalizeBucketRegion(region), tracker)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", bucket, err))
			continue
		}
		results = append(results, schema.BucketResult{
			Action:      "total",
			BucketName:  bucket,
			ObjectCount: int64(count),
			Message:     fmt.Sprintf("%d objects", count),
		})
	}
	if len(errs) > 0 {
		return results, errors.New(strings.Join(errs, "; "))
	}
	return results, nil
}

func (d *Driver) countBucketObjects(ctx context.Context, bucket, region string, tracker *processbar.CountTracker) (int, error) {
	client, err := d.NewClient()
	if err != nil {
		return 0, err
	}

	token := ""
	count := 0
	for {
		resp, err := client.ListObjectsV2(ctx, bucket, region, token, 1000)
		if err != nil {
			return 0, err
		}
		count += len(resp.Contents)
		if tracker != nil {
			tracker.Update(bucket, count)
		}
		if !resp.IsTruncated {
			return count, nil
		}
		token = strings.TrimSpace(resp.NextContinuationToken)
		if token == "" {
			return 0, fmt.Errorf("volcengine tos: missing next continuation token for bucket %s", bucket)
		}
	}
}

func normalizeBucketRegion(region string) string {
	region = strings.TrimSpace(region)
	if region == "" || region == "all" {
		return ""
	}
	return region
}
