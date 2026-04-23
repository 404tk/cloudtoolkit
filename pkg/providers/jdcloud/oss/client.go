package oss

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	awsapi "github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	awsauth "github.com/404tk/cloudtoolkit/pkg/providers/aws/auth"
	jdauth "github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/auth"
)

type Client struct {
	api *awsapi.Client
}

func NewClient(credential jdauth.Credential, opts ...awsapi.Option) *Client {
	awsCredential := awsauth.New(credential.AccessKey, credential.SecretKey, credential.SessionToken)
	return &Client{api: awsapi.NewClient(awsCredential, opts...)}
}

func (c *Client) ListObjectsV2(ctx context.Context, bucket, region, continuationToken string, maxKeys int) (ListObjectsV2Output, error) {
	if c == nil || c.api == nil {
		return ListObjectsV2Output{}, errors.New("jdcloud oss: nil object client")
	}

	query := url.Values{}
	query.Set("list-type", "2")
	if continuationToken = strings.TrimSpace(continuationToken); continuationToken != "" {
		query.Set("continuation-token", continuationToken)
	}
	if maxKeys > 0 {
		query.Set("max-keys", strconv.Itoa(maxKeys))
	}

	var wire listObjectsV2Response
	err := c.api.DoRESTXML(ctx, awsapi.Request{
		Service:    "s3",
		Region:     normalizeBucketRegion(region),
		Method:     http.MethodGet,
		Path:       bucketPath(bucket),
		Query:      query,
		Host:       serviceHost(region),
		Idempotent: true,
	}, &wire)
	if err != nil {
		return ListObjectsV2Output{}, err
	}

	out := ListObjectsV2Output{
		Objects:               make([]Object, 0, len(wire.Contents)),
		IsTruncated:           wire.IsTruncated,
		NextContinuationToken: strings.TrimSpace(wire.NextContinuationToken),
	}
	for _, object := range wire.Contents {
		key := strings.TrimSpace(object.Key)
		if key == "" {
			continue
		}
		out.Objects = append(out.Objects, Object{
			Key:  key,
			Size: object.Size,
		})
	}
	return out, nil
}
