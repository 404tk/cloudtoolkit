package oss

import (
	"context"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type BucketProvider struct {
	Client *oss.Client
}

func (d *BucketProvider) GetBuckets(ctx context.Context) ([]*schema.Storage, error) {
	list := schema.NewResources().Storages
	select {
	case <-ctx.Done():
		return list, nil
	default:
		log.Println("[*] Start enumerating OSS ...")
	}
	response, err := d.Client.ListBuckets(oss.MaxKeys(1000))
	if err != nil {
		log.Println("[-] Enumerate OSS failed.")
		return list, err
	}

	for _, bucket := range response.Buckets {
		/*
			if !strings.Contains(d.Client.Config.Endpoint, bucket.Location) {
				continue
			}
		*/
		_bucket := &schema.Storage{
			BucketName: bucket.Name,
			Region:     bucket.Location,
		}
		list = append(list, _bucket)
	}

	return list, nil
}
