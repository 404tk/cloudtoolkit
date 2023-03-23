package cos

import (
	"context"
	"log"
	"net/http"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentyun/cos-go-sdk-v5"
)

type COSProvider struct {
	Credential *common.Credential
}

func (d *COSProvider) GetBuckets(ctx context.Context) ([]*schema.Storage, error) {
	list := schema.NewResources().Storages
	select {
	case <-ctx.Done():
		return list, nil
	default:
		log.Println("[*] Start enumerating COS ...")
	}
	client := cos.NewClient(nil, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  d.Credential.SecretId,
			SecretKey: d.Credential.SecretKey,
		},
	})
	buckets, _, err := client.Service.Get(ctx)
	if err != nil {
		log.Println("[-] Enumerate COS failed.")
		return nil, err
	}

	for _, bucket := range buckets.Buckets {
		_bucket := &schema.Storage{
			BucketName: bucket.Name,
			Region:     bucket.Region,
		}

		list = append(list, _bucket)
	}

	return list, nil
}
