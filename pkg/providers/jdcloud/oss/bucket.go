package oss

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	jdauth "github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/auth"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const defaultBucketRegion = "cn-north-1"

var knownJDCloudOSSRegions = []string{
	"cn-north-1",
	"cn-east-1",
	"cn-east-2",
	"cn-south-1",
	"eu-west-1",
}

type Driver struct {
	Client       *api.Client
	Credential   jdauth.Credential
	Region       string
	ObjectClient *Client
}

func (d *Driver) ListBuckets(ctx context.Context) ([]schema.Storage, error) {
	list := []schema.Storage{}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List OSS buckets ...")
	}
	if d.Client == nil {
		return list, errors.New("jdcloud oss: nil api client")
	}

	var resp api.ListBucketsResponse
	err := d.Client.DoJSON(ctx, api.Request{
		Service: "oss",
		Region:  "cn-north-1",
		Method:  "GET",
		Version: "v1",
		Path:    "/regions/cn-north-1/buckets",
	}, &resp)
	if err != nil {
		logger.Error("List buckets failed.")
		return list, err
	}

	for _, bucket := range resp.Result.Buckets {
		_bucket := schema.Storage{
			BucketName: bucket.Name,
		}
		if region, err := d.ResolveBucketRegion(ctx, bucket.Name); err == nil {
			_bucket.Region = region
		}
		list = append(list, _bucket)
	}

	return list, nil
}

func (d *Driver) ResolveBucketRegion(ctx context.Context, bucket string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return "", fmt.Errorf("empty bucket name")
	}
	if d.Client == nil {
		return "", errors.New("jdcloud oss: nil api client")
	}

	var lastErr error
	for _, region := range d.probeRegions() {
		err := d.headBucket(ctx, region, bucket)
		if err == nil {
			return region, nil
		}
		lastErr = err
		var apiErr *api.APIError
		if errors.As(err, &apiErr) && apiErr.IsAuthFailure() {
			return "", err
		}
	}
	if lastErr != nil {
		return "", fmt.Errorf("bucket %s region not found", bucket)
	}
	return "", fmt.Errorf("bucket %s region not found", bucket)
}

func (d *Driver) objectClient() (*Client, error) {
	if d.ObjectClient != nil {
		return d.ObjectClient, nil
	}
	if err := d.Credential.Validate(); err != nil {
		return nil, err
	}
	d.ObjectClient = NewClient(d.Credential)
	return d.ObjectClient, nil
}

func (d *Driver) headBucket(ctx context.Context, region, bucket string) error {
	return d.Client.DoJSON(ctx, api.Request{
		Service: "oss",
		Region:  region,
		Method:  http.MethodHead,
		Version: "v1",
		Path:    "/regions/" + region + "/buckets/" + strings.TrimSpace(bucket),
	}, nil)
}

func (d *Driver) probeRegions() []string {
	explicit := d.normalizedRegion()
	if explicit == "" || explicit == "all" {
		return append([]string(nil), knownJDCloudOSSRegions...)
	}

	regions := []string{explicit}
	for _, region := range knownJDCloudOSSRegions {
		if strings.EqualFold(region, explicit) {
			continue
		}
		regions = append(regions, region)
	}
	return regions
}

func (d *Driver) normalizedRegion() string {
	region := strings.TrimSpace(d.Region)
	if strings.EqualFold(region, "all") {
		return "all"
	}
	return region
}
