package tos

import (
	"context"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Cred          auth.Credential
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
		logger.Info("List TOS buckets ...")
	}

	client, err := d.NewClient()
	if err != nil {
		return list, err
	}
	resp, err := client.ListBuckets(ctx, d.Region)
	if err != nil {
		logger.Error("List buckets failed.")
		return list, err
	}
	for _, bucket := range resp.Buckets {
		list = append(list, schema.Storage{
			BucketName: bucket.Name,
			Region:     strings.TrimSpace(bucket.Location),
		})
	}
	return list, nil
}
