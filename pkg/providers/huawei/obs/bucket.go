package obs

import (
	"context"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
)

type OBSProvider struct {
	Auth    basic.Credentials
	Regions []string
}

func (d *OBSProvider) GetBuckets(ctx context.Context) ([]*schema.Storage, error) {
	list := schema.NewResources().Storages
	log.Println("[*] Start enumerating OBS ...")
	endPoint := "obs." + d.Regions[0] + ".myhuaweicloud.com"
	client, err := obs.New(d.Auth.AK, d.Auth.SK, endPoint)
	if err != nil {
		log.Println("[-] Enumerate OBS failed.")
		return nil, err
	}

	response, err := client.ListBuckets(nil)
	if err != nil {
		log.Println("[-] Enumerate OBS failed.")
		return list, err
	}

	for _, bucket := range response.Buckets {
		_bucket := &schema.Storage{
			BucketName: bucket.Name,
			Region:     bucket.Location,
		}
		list = append(list, _bucket)
	}

	return list, nil
}
