package cos

import (
	"context"
	"net/http"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentyun/cos-go-sdk-v5"
)

type Driver struct {
	Credential *common.Credential
}

func (d *Driver) GetBuckets(ctx context.Context) ([]schema.Storage, error) {
	list := []schema.Storage{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("Start enumerating COS ...")
	}
	client := cos.NewClient(nil, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:     d.Credential.SecretId,
			SecretKey:    d.Credential.SecretKey,
			SessionToken: d.Credential.Token,
		},
	})
	buckets, _, err := client.Service.Get(ctx)
	if err != nil {
		logger.Error("Enumerate COS failed.")
		return nil, err
	}

	for _, bucket := range buckets.Buckets {
		_bucket := schema.Storage{
			BucketName: bucket.Name,
			Region:     bucket.Region,
		}

		list = append(list, _bucket)
	}

	return list, nil
}
