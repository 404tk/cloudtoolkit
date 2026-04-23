package oss

import (
	"context"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

func (d *Driver) ListObjects(ctx context.Context, buckets map[string]string) {
	for bucket, region := range buckets {
		resp, err := d.listObjectsPage(ctx, bucket, region, "", 100)
		if err != nil {
			logger.Error(fmt.Sprintf("List Objects in %s failed: %s", bucket, err))
			continue
		}

		if len(resp.Objects) == 0 {
			logger.Error(fmt.Sprintf("No Objects found in %s.", bucket))
			continue
		}
		logger.Warning(fmt.Sprintf("%d objects found in %s.", len(resp.Objects), bucket))

		fmt.Printf("\n%-70s\t%-10s\n", "Key", "Size")
		fmt.Printf("%-70s\t%-10s\n", "---", "----")
		for _, object := range resp.Objects {
			fmt.Printf("%-70s\t%-10s\n", object.Key, utils.ParseBytes(object.Size))
		}
		fmt.Println()

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func (d *Driver) TotalObjects(ctx context.Context, buckets map[string]string) {
	tracker := processbar.NewCountTracker()
	defer tracker.Finish()

	for bucket, region := range buckets {
		count, err := d.countBucketObjects(ctx, bucket, region, tracker)
		if err != nil {
			logger.Error(fmt.Sprintf("List Objects in %s failed: %s", bucket, err))
			return
		}
		fmt.Printf("\r")
		logger.Warning(fmt.Sprintf("%s has %d objects.", bucket, count))
	}
}

func (d *Driver) listObjectsPage(ctx context.Context, bucket, region, token string, maxKeys int) (ListObjectsV2Output, error) {
	client, err := d.objectClient()
	if err != nil {
		return ListObjectsV2Output{}, err
	}
	return client.ListObjectsV2(ctx, bucket, region, token, maxKeys)
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
