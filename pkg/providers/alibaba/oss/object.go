package oss

import (
	"context"
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

	results := []schema.BucketResult{}
	for b, r := range buckets {
		resp, err := client.ListObjectsV2(ctx, b, normalizeBucketRegion(r), "", 100)
		if err != nil {
			return nil, fmt.Errorf("list objects in %s: %w", b, err)
		}

		objects := make([]schema.BucketObject, 0, len(resp.Objects))
		for _, obj := range resp.Objects {
			objects = append(objects, schema.BucketObject{
				BucketName:   b,
				Key:          obj.Key,
				Size:         obj.Size,
				LastModified: obj.LastModified,
				StorageClass: obj.StorageClass,
			})
		}

		result := schema.BucketResult{
			Action:      "list",
			BucketName:  b,
			ObjectCount: int64(len(objects)),
			Objects:     objects,
			Message:     fmt.Sprintf("%d objects found", len(objects)),
		}
		results = append(results, result)

		select {
		case <-ctx.Done():
			return results, nil
		default:
		}
	}
	return results, nil
}

/*
Recommended：

	./ossutil64 du oss://examplebucket/dir/ --block-size GB

Links:

	https://help.aliyun.com/document_detail/129732.html
	https://github.com/aliyun/ossutil
*/
func (d *Driver) TotalObjects(ctx context.Context, buckets map[string]string) ([]schema.BucketResult, error) {
	tracker := processbar.NewCountTracker()
	defer tracker.Finish()

	results := []schema.BucketResult{}
	for b, r := range buckets {
		count, err := d.countBucketObjects(ctx, b, normalizeBucketRegion(r), tracker)
		if err != nil {
			return nil, fmt.Errorf("list objects in %s: %w", b, err)
		}

		result := schema.BucketResult{
			Action:      "total",
			BucketName:  b,
			ObjectCount: int64(count),
			Message:     fmt.Sprintf("%d objects", count),
		}
		results = append(results, result)
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
		count += len(resp.Objects)
		select {
		case <-ctx.Done():
			return count, nil
		default:
			if tracker != nil {
				tracker.Update(bucket, count)
			}
		}
		if !resp.IsTruncated {
			return count, nil
		}
		token = strings.TrimSpace(resp.NextContinuationToken)
		if token == "" {
			return 0, fmt.Errorf("alibaba oss: missing next continuation token for bucket %s", bucket)
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
