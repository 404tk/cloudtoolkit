package obs

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

func (d *Driver) ListObjects(ctx context.Context, buckets map[string]string) ([]schema.BucketResult, error) {
	results := make([]schema.BucketResult, 0, len(buckets))
	var errs []string
	for bucket, region := range buckets {
		marker := ""
		objects := make([]schema.BucketObject, 0)
		failed := false
		for {
			resp, err := d.listObjectsPage(ctx, bucket, region, marker, 100)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", bucket, err))
				failed = true
				break
			}

			for _, obj := range resp.Objects {
				objects = append(objects, schema.BucketObject{
					BucketName:   bucket,
					Key:          obj.Key,
					Size:         obj.Size,
					LastModified: obj.LastModified,
					StorageClass: obj.StorageClass,
				})
			}

			if !resp.IsTruncated {
				break
			}
			marker = nextMarker(resp)

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
		count, err := d.countBucketObjects(ctx, bucket, region, tracker)
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

func (d *Driver) listObjectsPage(ctx context.Context, bucket, region, marker string, maxKeys int) (ListObjectsResponse, error) {
	client := d.client()
	if client == nil {
		return ListObjectsResponse{}, fmt.Errorf("huawei obs: nil client")
	}
	return client.ListObjects(ctx, bucket, d.resolveBucketRegion(region), marker, maxKeys)
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
			return 0, fmt.Errorf("huawei obs: truncated response for bucket %s missing continuation marker", bucket)
		}
		marker = next
	}
}

func (d *Driver) resolveBucketRegion(region string) string {
	region = strings.TrimSpace(region)
	if region != "" && region != "all" {
		return region
	}
	return d.requestRegion()
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
