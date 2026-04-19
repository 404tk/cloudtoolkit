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
	client, err := d.NewClient()
	if err != nil {
		logger.Error(err)
		return
	}
	for b, r := range buckets {
		resp, err := client.ListObjectsV2(ctx, b, normalizeBucketRegion(r), "", 100)
		if err != nil {
			msg := fmt.Sprintf("List Objects in %s failed: %s", b, err.Error())
			logger.Error(msg)
			continue
		}

		if len(resp.Objects) == 0 {
			msg := fmt.Sprintf("No Objects found in %s.", b)
			logger.Error(msg)
			continue
		}
		logger.Warning(fmt.Sprintf("%d objects found in %s.", len(resp.Objects), b))

		fmt.Printf("\n%-70s\t%-10s\n", "Key", "Size")
		fmt.Printf("%-70s\t%-10s\n", "---", "----")
		for _, object := range resp.Objects {
			fmt.Printf("%-70s\t%-10s\n",
				object.Key, utils.ParseBytes(object.Size))
		}
		fmt.Println()
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

/*
Recommended：

	./ossutil64 du oss://examplebucket/dir/ --block-size GB

Links:

	https://help.aliyun.com/document_detail/129732.html
	https://github.com/aliyun/ossutil
*/
func (d *Driver) TotalObjects(ctx context.Context, buckets map[string]string) {
	tracker := processbar.NewCountTracker()
	defer tracker.Finish()
	for b, r := range buckets {
		count, err := d.countBucketObjects(ctx, b, normalizeBucketRegion(r), tracker)
		if err != nil {
			logger.Error(fmt.Sprintf("List Objects in %s failed: %s", b, err))
			return
		}
		fmt.Printf("\r")
		logger.Warning(fmt.Sprintf("%s has %d objects.", b, count))
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
