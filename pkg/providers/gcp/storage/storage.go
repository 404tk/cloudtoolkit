// Package storage wraps GCS bucket / object enumeration for cloudlist and
// bucket-check.
package storage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

const (
	defaultPageSize = 200
	maxPages        = 50
)

type Driver struct {
	Client   *api.Client
	Projects []string
}

// GetBuckets lists GCS buckets across the configured projects.
func (d *Driver) GetBuckets(ctx context.Context) ([]schema.Storage, error) {
	if d == nil || d.Client == nil {
		return nil, errors.New("gcp storage: nil api client")
	}
	out := make([]schema.Storage, 0)
	for _, project := range d.Projects {
		project = strings.TrimSpace(project)
		if project == "" {
			continue
		}
		buckets, err := d.listBuckets(ctx, project)
		if err != nil {
			return out, err
		}
		for _, b := range buckets {
			out = append(out, schema.Storage{
				BucketName: b.Name,
				Region:     b.Location,
			})
		}
	}
	return out, nil
}

func (d *Driver) listBuckets(ctx context.Context, project string) ([]api.GCSBucket, error) {
	out := make([]api.GCSBucket, 0)
	pageToken := ""
	for page := 0; page < maxPages; page++ {
		query := url.Values{}
		query.Set("project", project)
		query.Set("maxResults", strconv.Itoa(defaultPageSize))
		if pageToken != "" {
			query.Set("pageToken", pageToken)
		}
		var resp api.GCSBucketsListResponse
		if err := d.Client.Do(ctx, api.Request{
			Method:     http.MethodGet,
			BaseURL:    api.StorageBaseURL,
			Path:       "/storage/v1/b",
			Query:      query,
			Idempotent: true,
		}, &resp); err != nil {
			return out, err
		}
		out = append(out, resp.Items...)
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return out, nil
}

// ListObjects walks objects in `infos` (bucket → "" since GCS is global).
func (d *Driver) ListObjects(ctx context.Context, infos map[string]string) ([]schema.BucketResult, error) {
	results := make([]schema.BucketResult, 0, len(infos))
	for bucket := range infos {
		objects, err := d.listObjects(ctx, bucket)
		if err != nil {
			return results, err
		}
		results = append(results, schema.BucketResult{
			Action:      "list",
			BucketName:  bucket,
			ObjectCount: int64(len(objects)),
			Objects:     objects,
		})
	}
	return results, nil
}

func (d *Driver) listObjects(ctx context.Context, bucket string) ([]schema.BucketObject, error) {
	out := make([]schema.BucketObject, 0)
	pageToken := ""
	for page := 0; page < maxPages; page++ {
		query := url.Values{}
		query.Set("maxResults", strconv.Itoa(defaultPageSize))
		if pageToken != "" {
			query.Set("pageToken", pageToken)
		}
		var resp api.GCSObjectsListResponse
		if err := d.Client.Do(ctx, api.Request{
			Method:     http.MethodGet,
			BaseURL:    api.StorageBaseURL,
			Path:       fmt.Sprintf("/storage/v1/b/%s/o", url.PathEscape(bucket)),
			Query:      query,
			Idempotent: true,
		}, &resp); err != nil {
			return out, err
		}
		for _, item := range resp.Items {
			size, _ := strconv.ParseInt(item.Size, 10, 64)
			out = append(out, schema.BucketObject{
				BucketName:   item.Bucket,
				Key:          item.Name,
				Size:         size,
				LastModified: item.Updated,
				StorageClass: item.StorageClass,
			})
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return out, nil
}
