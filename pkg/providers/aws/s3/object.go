package s3

import (
	"context"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

func (d *Driver) ListObjects(ctx context.Context, buckets map[string]string) ([]schema.BucketResult, error) {
	results := []schema.BucketResult{}
	for b, r := range buckets {
		resp, err := d.listObjectsPage(ctx, b, r, "", 100)
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

func (d *Driver) TotalObjects(ctx context.Context, buckets map[string]string) ([]schema.BucketResult, error) {
	tracker := processbar.NewCountTracker()
	defer tracker.Finish()

	results := []schema.BucketResult{}
	for b, r := range buckets {
		count, err := d.countBucketObjects(ctx, b, r, tracker)
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

func (d *Driver) listObjectsPage(ctx context.Context, bucket, region, token string, maxKeys int) (api.ListObjectsV2Output, error) {
	client, err := d.requireClient()
	if err != nil {
		return api.ListObjectsV2Output{}, err
	}
	return client.ListObjectsV2(ctx, d.resolveRegion(region), bucket, token, maxKeys)
}

func (d *Driver) countBucketObjects(ctx context.Context, bucket, region string, tracker *processbar.CountTracker) (int, error) {
	count := 0
	token := ""
	for {
		resp, err := d.listObjectsPage(ctx, bucket, region, token, 1000)
		if err != nil {
			return 0, err
		}
		count += len(resp.Objects)
		if tracker != nil {
			tracker.Update(bucket, count)
		}
		if !resp.IsTruncated || strings.TrimSpace(resp.NextContinuationToken) == "" {
			return count, nil
		}
		token = resp.NextContinuationToken
	}
}

func (d *Driver) resolveRegion(region string) string {
	region = strings.TrimSpace(region)
	if region != "" {
		return region
	}
	return d.defaultRegion()
}
