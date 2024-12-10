package oss

import (
	"context"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/jdcloud-api/jdcloud-sdk-go/core"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/oss/apis"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/oss/client"
)

type Driver struct {
	Cred  *core.Credential
	Token string
}

func (d *Driver) newClient() *client.OssClient {
	c := client.NewOssClient(d.Cred)
	c.SetLogger(core.NewDummyLogger())
	return c
}

func (d *Driver) ListBuckets(ctx context.Context) ([]schema.Storage, error) {
	list := []schema.Storage{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List OSS buckets ...")
	}
	svc := d.newClient()
	req := apis.NewListBucketsRequest("cn-north-1")
	req.AddHeader("x-jdcloud-security-token", d.Token)
	resp, err := svc.ListBuckets(req)
	if err != nil {
		logger.Error("List buckets failed.")
		return list, err
	}

	for _, bucket := range resp.Result.Buckets {
		_bucket := schema.Storage{
			BucketName: bucket.Name,
		}
		list = append(list, _bucket)
	}

	return list, nil
}
