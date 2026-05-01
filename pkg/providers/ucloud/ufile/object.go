package ufile

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const (
	listObjectsLimit = 1000
)

// fileClient is the per-bucket UFile client. Construction is lazy so the
// existing JSON-RPC `Client` injection pattern is preserved.
func (d *Driver) fileClient() *FileClient {
	if d.FileClient != nil {
		return d.FileClient
	}
	return NewFileClient(d.Credential)
}

// ListObjects fans out across the resolved bucket→region map and returns one
// schema.BucketResult per bucket containing the page of objects discovered.
// Mirrors the alibaba/aws/tencent BucketDump shape.
func (d *Driver) ListObjects(ctx context.Context, buckets map[string]string) ([]schema.BucketResult, error) {
	out := make([]schema.BucketResult, 0, len(buckets))
	if len(buckets) == 0 {
		return out, nil
	}
	logger.Info("List UFile objects ...")
	client := d.fileClient()
	for bucket, region := range buckets {
		bucket = strings.TrimSpace(bucket)
		region = strings.TrimSpace(region)
		if bucket == "" {
			continue
		}
		if region == "" {
			region = strings.TrimSpace(d.Region)
		}
		resp, err := client.PrefixFileList(ctx, bucket, region, "", "", listObjectsLimit)
		result := schema.BucketResult{
			Action:     "list",
			BucketName: bucket,
		}
		if err != nil {
			result.Message = err.Error()
			out = append(out, result)
			continue
		}
		objects := make([]schema.BucketObject, 0, len(resp.DataSet))
		for _, item := range resp.DataSet {
			objects = append(objects, schema.BucketObject{
				BucketName:   bucket,
				Key:          item.FileName,
				Size:         item.Size,
				LastModified: formatModifyTime(item.ModifyTime),
				StorageClass: item.StorageClass,
			})
		}
		result.Objects = objects
		result.ObjectCount = int64(len(objects))
		out = append(out, result)
	}
	return out, nil
}

// TotalObjects pages through every bucket and returns the aggregate object
// count + total size. The summary view is what the bucket-check `total`
// action surfaces in the REPL table.
func (d *Driver) TotalObjects(ctx context.Context, buckets map[string]string) ([]schema.BucketResult, error) {
	out := make([]schema.BucketResult, 0, len(buckets))
	if len(buckets) == 0 {
		return out, nil
	}
	logger.Info("Total UFile objects ...")
	client := d.fileClient()
	for bucket, region := range buckets {
		bucket = strings.TrimSpace(bucket)
		region = strings.TrimSpace(region)
		if bucket == "" {
			continue
		}
		if region == "" {
			region = strings.TrimSpace(d.Region)
		}
		var (
			count   int64
			size    int64
			marker  string
			lastErr error
		)
		for attempt := 0; attempt < 200; attempt++ {
			resp, err := client.PrefixFileList(ctx, bucket, region, "", marker, listObjectsLimit)
			if err != nil {
				lastErr = err
				break
			}
			for _, item := range resp.DataSet {
				count++
				size += item.Size
			}
			if strings.TrimSpace(resp.NextMarker) == "" || len(resp.DataSet) == 0 {
				break
			}
			marker = resp.NextMarker
		}
		result := schema.BucketResult{
			Action:      "total",
			BucketName:  bucket,
			ObjectCount: count,
		}
		if lastErr != nil {
			result.Message = lastErr.Error()
		} else {
			result.Message = fmt.Sprintf("%d objects, %d bytes", count, size)
		}
		out = append(out, result)
	}
	return out, nil
}

// ResolveBucketRegion looks up the region for a single bucket via DescribeBucket.
// Used by ucloud.Provider.bucketInfos when the caller passes a specific bucket.
func (d *Driver) ResolveBucketRegion(ctx context.Context, bucket string) (string, error) {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return "", fmt.Errorf("ucloud ufile: empty bucket")
	}
	storages, err := d.GetBuckets(ctx)
	if err != nil {
		return "", err
	}
	for _, s := range storages {
		if s.BucketName == bucket && s.Region != "" {
			return s.Region, nil
		}
	}
	return "", fmt.Errorf("ucloud ufile: region for bucket %q not found", bucket)
}

func formatModifyTime(unix int64) string {
	if unix <= 0 {
		return ""
	}
	return time.Unix(unix, 0).UTC().Format(time.RFC3339)
}
