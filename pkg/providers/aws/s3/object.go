package s3

import (
	"context"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

func (d *Driver) ListObjects(ctx context.Context, buckets map[string]string) {
	for b, r := range buckets {
		resp, err := d.listObjectsPage(ctx, b, r, "", 100)
		if err != nil {
			logger.Error(fmt.Sprintf("List Objects in %s failed: %s", b, err.Error()))
			continue
		}

		if len(resp.Objects) == 0 {
			logger.Error(fmt.Sprintf("No Objects found in %s.", b))
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

func (d *Driver) TotalObjects(ctx context.Context, buckets map[string]string) {
	tracker := processbar.NewCountTracker()
	defer tracker.Finish()
	for b, r := range buckets {
		count, err := d.countBucketObjects(ctx, b, r, tracker)
		if err != nil {
			logger.Error(fmt.Sprintf("List Objects in %s failed: %s", b, err))
			return
		}
		fmt.Printf("\r")
		logger.Warning(fmt.Sprintf("%s has %d objects.", b, count))
	}
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
