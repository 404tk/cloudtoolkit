package oss

import (
	"context"
	"fmt"
	"log"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

func (d *Driver) ListObjects(ctx context.Context, buckets map[string]string) {
	for b, r := range buckets {
		d.Region = r
		client := d.NewClient()
		bucket, err := client.Bucket(b)
		if err != nil {
			log.Println("[-]", err)
			return
		}
		resp, err := bucket.ListObjectsV2(oss.MaxKeys(100))
		if err != nil {
			log.Printf("[-] List Objects in %s failed: %s\n", b, err.Error())
			continue
		}

		if len(resp.Objects) == 0 {
			log.Printf("[-] No Objects found in %s.\n", b)
			continue
		}
		log.Printf("[+] %d objects found in %s.\n", len(resp.Objects), b)

		fmt.Printf("\n%-70s\t%-10s\n", "Key", "Size")
		fmt.Printf("%-70s\t%-10s\n", "---", "----")
		for _, object := range resp.Objects {
			fmt.Printf("%-70s\t%-10s\n",
				object.Key, fmt.Sprintf("%v bytes", object.Size))
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
	for b, r := range buckets {
		var token string
		count := 0
		isTruncated := true
		for isTruncated {
			d.Region = r
			client := d.NewClient()
			bucket, err := client.Bucket(b)
			if err != nil {
				log.Println("[-]", err)
				return
			}
			resp, err := bucket.ListObjectsV2(oss.MaxKeys(1000), oss.ContinuationToken(token))
			if err != nil {
				log.Printf("[-] List Objects in %s failed: %s\n", b, err)
				continue
			}

			isTruncated = resp.IsTruncated
			token = resp.NextContinuationToken
			count += len(resp.Objects)
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
		log.Printf("[+] %s has %d objects.\n", b, count)
	}
}
