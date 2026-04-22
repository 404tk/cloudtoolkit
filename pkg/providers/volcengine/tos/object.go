package tos

import (
	"context"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

func (d *Driver) ListObjects(ctx context.Context, buckets map[string]string) {
	client, err := d.NewClient()
	if err != nil {
		logger.Error(err)
		return
	}

	for bucket, region := range buckets {
		resp, err := client.ListObjectsV2(ctx, bucket, normalizeBucketRegion(region), "", 100)
		if err != nil {
			logger.Error(fmt.Sprintf("List Objects in %s failed: %s", bucket, err))
			continue
		}
		if len(resp.Contents) == 0 {
			logger.Error(fmt.Sprintf("No Objects found in %s.", bucket))
			continue
		}
		logger.Warning(fmt.Sprintf("%d objects found in %s.", len(resp.Contents), bucket))
		fmt.Printf("\n%-70s\t%-10s\n", "Key", "Size")
		fmt.Printf("%-70s\t%-10s\n", "---", "----")
		for _, object := range resp.Contents {
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
		count, err := d.countBucketObjects(ctx, bucket, normalizeBucketRegion(region), tracker)
		if err != nil {
			logger.Error(fmt.Sprintf("List Objects in %s failed: %s", bucket, err))
			return
		}
		fmt.Printf("\r")
		logger.Warning(fmt.Sprintf("%s has %d objects.", bucket, count))
	}
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
