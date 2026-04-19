package oss

import (
	"context"
	"strings"

	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Cred          aliauth.Credential
	Region        string
	Client        *Client
	clientOptions []Option
}

func (d *Driver) NewClient() (*Client, error) {
	if err := d.Cred.Validate(); err != nil {
		return nil, err
	}
	if d.Client != nil {
		return d.Client, nil
	}
	return NewClient(d.Cred, d.clientOptions...), nil
}

func (d *Driver) GetBuckets(ctx context.Context) ([]schema.Storage, error) {
	list := []schema.Storage{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List OSS buckets ...")
	}
	client, err := d.NewClient()
	if err != nil {
		return list, err
	}
	response, err := client.ListBuckets(ctx, d.Region)
	if err != nil {
		logger.Error("List buckets failed.")
		return list, err
	}

	for _, bucket := range response.Buckets {
		region := strings.TrimSpace(bucket.Region)
		if region == "" {
			region = strings.TrimPrefix(bucket.Location, "oss-")
		}
		_bucket := schema.Storage{
			BucketName: bucket.Name,
			Region:     region,
		}
		list = append(list, _bucket)
	}

	return list, nil
}
