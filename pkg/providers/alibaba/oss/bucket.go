package oss

import (
	"context"
	"strings"

	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type Driver struct {
	Cred          aliauth.Credential
	Region        string
	clientOptions []oss.ClientOption
}

func (d *Driver) NewClient() (*oss.Client, error) {
	if err := d.Cred.Validate(); err != nil {
		return nil, err
	}
	region := d.Region
	if region == "all" {
		region = "cn-hangzhou"
	}
	options := append([]oss.ClientOption{}, d.clientOptions...)
	if d.Cred.SecurityToken != "" {
		options = append(options, oss.SecurityToken(d.Cred.SecurityToken))
	}
	return oss.New(
		"https://oss-"+region+".aliyuncs.com",
		d.Cred.AccessKeyID,
		d.Cred.AccessKeySecret,
		options...)
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
	response, err := client.ListBuckets(oss.MaxKeys(1000))
	if err != nil {
		logger.Error("List buckets failed.")
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
			Region:     strings.TrimPrefix(bucket.Location, "oss-"),
		}
		list = append(list, _bucket)
	}

	return list, nil
}
