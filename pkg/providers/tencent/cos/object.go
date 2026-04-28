package cos

import (
	"context"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

func (d *Driver) ListObjects(ctx context.Context, buckets map[string]string) ([]schema.BucketResult, error) {
	results := []schema.BucketResult{}
	for bucket, region := range buckets {
		resp, err := d.listObjectsPage(ctx, bucket, region, "", 100)
		if err != nil {
			return nil, fmt.Errorf("list objects in %s: %w", bucket, err)
		}

		objects := make([]schema.BucketObject, 0, len(resp.Objects))
		for _, obj := range resp.Objects {
			objects = append(objects, schema.BucketObject{
				BucketName:   bucket,
				Key:          obj.Key,
				Size:         obj.Size,
				LastModified: obj.LastModified,
				StorageClass: obj.StorageClass,
			})
		}

		result := schema.BucketResult{
			Action:      "list",
			BucketName:  bucket,
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
	for bucket, region := range buckets {
		count, err := d.countBucketObjects(ctx, bucket, region, tracker)
		if err != nil {
			return nil, fmt.Errorf("list objects in %s: %w", bucket, err)
		}

		result := schema.BucketResult{
			Action:      "total",
			BucketName:  bucket,
			ObjectCount: int64(count),
			Message:     fmt.Sprintf("%d objects", count),
		}
		results = append(results, result)
	}
	return results, nil
}

func (d *Driver) listObjectsPage(ctx context.Context, bucket, region, marker string, maxKeys int) (ListObjectsResponse, error) {
	client := d.client()
	if client == nil {
		return ListObjectsResponse{}, fmt.Errorf("tencent cos: nil client")
	}
	return client.ListObjects(ctx, bucket, normalizeRegion(region), marker, maxKeys)
}

func (d *Driver) countBucketObjects(ctx context.Context, bucket, region string, tracker *processbar.CountTracker) (int, error) {
	count := 0
	marker := ""
	for {
		resp, err := d.listObjectsPage(ctx, bucket, region, marker, 1000)
		if err != nil {
			return 0, err
		}
		count += len(resp.Objects)
		if tracker != nil {
			tracker.Update(bucket, count)
		}
		if !resp.IsTruncated {
			return count, nil
		}

		next := nextMarker(resp)
		if next == "" {
			return 0, fmt.Errorf("tencent cos: truncated response for bucket %s missing continuation marker", bucket)
		}
		marker = next
	}
}

func nextMarker(resp ListObjectsResponse) string {
	if marker := strings.TrimSpace(resp.NextMarker); marker != "" {
		return marker
	}
	if len(resp.Objects) == 0 {
		return ""
	}
	return strings.TrimSpace(resp.Objects[len(resp.Objects)-1].Key)
}

func normalizeRegion(region string) string {
	region = strings.TrimSpace(region)
	if region == "" || region == "all" {
		return ""
	}
	return region
}
