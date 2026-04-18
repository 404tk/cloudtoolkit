package s3

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func (d *Driver) ListObjects(ctx context.Context, buckets map[string]string) {
	for b, r := range buckets {
		client := d.clientForRegion(r)
		limit := int32(100) // Do not display more yet.
		input := &s3.ListObjectsV2Input{Bucket: &b, MaxKeys: &limit}
		resp, err := client.ListObjectsV2(ctx, input)
		if err != nil {
			logger.Error(fmt.Sprintf("List Objects in %s failed: %s", b, err.Error()))
			continue
		}

		if len(resp.Contents) == 0 {
			logger.Error(fmt.Sprintf("No Objects found in %s.", b))
			continue
		}
		logger.Warning(fmt.Sprintf("%d objects found in %s.", len(resp.Contents), b))

		fmt.Printf("\n%-70s\t%-10s\n", "Key", "Size")
		fmt.Printf("%-70s\t%-10s\n", "---", "----")
		for _, object := range resp.Contents {
			size := int64(0)
			if object.Size != nil {
				size = *object.Size
			}
			fmt.Printf("%-70s\t%-10s\n",
				awsv2.ToString(object.Key), utils.ParseBytes(size))
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
		var token *string
		count := 0
		isTruncated := true
		client := d.clientForRegion(r)
		for isTruncated {
			limit := int32(1000)
			input := &s3.ListObjectsV2Input{
				Bucket:            &b,
				MaxKeys:           &limit,
				ContinuationToken: token,
			}
			resp, err := client.ListObjectsV2(ctx, input)
			if err != nil {
				logger.Error(fmt.Sprintf("List Objects in %s failed: %s", b, err))
				return
			}

			isTruncated = awsv2.ToBool(resp.IsTruncated)
			token = resp.NextContinuationToken
			count += len(resp.Contents)
			select {
			case <-ctx.Done():
				return
			default:
				tracker.Update(b, count)
			}
		}
		fmt.Printf("\r")
		logger.Warning(fmt.Sprintf("%s has %d objects.", b, count))
	}
}

func (d *Driver) clientForRegion(region string) *s3.Client {
	cfg := d.Config.Copy()
	if region != "" {
		cfg.Region = region
	}
	return s3.NewFromConfig(cfg)
}
