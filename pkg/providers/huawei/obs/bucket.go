package obs

import (
	"context"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Cred    auth.Credential
	Regions []string
	Client  *Client
}

func (d *Driver) client() *Client {
	if d.Client == nil {
		d.Client = NewClient(d.Cred)
	}
	return d.Client
}

func (d *Driver) GetBuckets(ctx context.Context) ([]schema.Storage, error) {
	list := []schema.Storage{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List OBS buckets...")
	}

	endpointRegion := d.requestRegion()
	resp, err := d.client().ListBuckets(ctx, endpointRegion)
	if err != nil {
		logger.Error("List buckets failed with", endpointRegion)
		return list, err
	}

	for _, bucket := range resp.Buckets {
		item := schema.Storage{
			BucketName: bucket.Name,
			Region:     bucket.Location,
		}
		if item.Region == "" {
			item.Region = endpointRegion
		}
		list = append(list, item)
	}
	return list, nil
}

func (d *Driver) requestRegion() string {
	if region := strings.TrimSpace(d.Cred.Region); region != "" && region != "all" {
		return region
	}
	for _, region := range d.Regions {
		region = strings.TrimSpace(region)
		if region != "" && region != "all" {
			return region
		}
	}
	return "cn-north-4"
}
