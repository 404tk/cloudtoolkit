package s3

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/utils"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
	"github.com/aws/aws-sdk-go/service/s3"
)

func (d *Driver) ListObjects(ctx context.Context, buckets map[string]string) {
	for b, r := range buckets {
		d.Session.Config.Region = &r
		client := s3.New(d.Session)
		var limit = int64(100) // Do not display more yet.
		input := &s3.ListObjectsV2Input{Bucket: &b, MaxKeys: &limit}
		resp, err := client.ListObjectsV2(input)
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
			fmt.Printf("%-70s\t%-10s\n",
				*object.Key, utils.ParseBytes(*object.Size))
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
	prevLength := 0
	for b, r := range buckets {
		var token *string
		count := 0
		isTruncated := true
		for isTruncated {
			d.Session.Config.Region = &r
			client := s3.New(d.Session)
			limit := int64(1000)
			input := &s3.ListObjectsV2Input{
				Bucket:            &b,
				MaxKeys:           &limit,
				ContinuationToken: token,
			}
			resp, err := client.ListObjectsV2(input)
			if err != nil {
				logger.Error(fmt.Sprintf("List Objects in %s failed: %s", b, err))
				return
			}

			isTruncated = *resp.IsTruncated
			token = resp.NextContinuationToken
			count += len(resp.Contents)
			select {
			case <-ctx.Done():
				return
			default:
				prevLength = processbar.CountPrint(b, count, prevLength)
			}
		}
		fmt.Printf("\r")
		logger.Warning(fmt.Sprintf("%s has %d objects.", b, count))
	}
}
