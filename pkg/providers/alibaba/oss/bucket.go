package oss

import (
	"context"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type Driver struct {
	Cred   *credentials.StsTokenCredential
	Region string
}

func (d *Driver) GetBuckets(ctx context.Context) ([]schema.Storage, error) {
	list := schema.NewResources().Storages
	select {
	case <-ctx.Done():
		return list, nil
	default:
		log.Println("[*] Start enumerating OSS ...")
	}
	region := d.Region
	if region == "all" {
		region = "cn-hangzhou"
	}
	client, err := oss.New(
		"oss-"+region+".aliyuncs.com",
		d.Cred.AccessKeyId,
		d.Cred.AccessKeySecret,
		oss.SecurityToken(d.Cred.AccessKeyStsToken))
	if err != nil {
		return list, err
	}
	response, err := client.ListBuckets(oss.MaxKeys(1000))
	if err != nil {
		log.Println("[-] List buckets failed.")
		return list, err
	}

	for _, bucket := range response.Buckets {
		/*
			if !strings.Contains(d.Client.Config.Endpoint, bucket.Location) {
				continue
			}
		*/
		_bucket := schema.Storage{
			BucketName: bucket.Name,
			Region:     bucket.Location,
		}
		list = append(list, _bucket)
	}

	return list, nil
}
