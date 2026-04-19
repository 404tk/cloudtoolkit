package vm

import (
	"context"
	"errors"
	"net/url"
	"strconv"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const vmPageSize = 100

type pageCursor struct {
	PageNumber int
	Seen       int
}

type Driver struct {
	Client *api.Client
	Region string
}

func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := []schema.Host{}
	if ctx == nil {
		ctx = context.Background()
	}
	logger.Info("List VM instances ...")
	if d.Client == nil {
		return list, errors.New("jdcloud vm: nil api client")
	}

	region := d.requestRegion()

	got, err := paginate.Fetch[schema.Host, pageCursor](ctx, func(ctx context.Context, cursor pageCursor) (paginate.Page[schema.Host, pageCursor], error) {
		pageNumber := cursor.PageNumber
		if pageNumber <= 0 {
			pageNumber = 1
		}

		query := url.Values{}
		query.Set("pageNumber", strconv.Itoa(pageNumber))
		query.Set("pageSize", strconv.Itoa(vmPageSize))

		var resp api.DescribeInstancesResponse
		err := d.Client.DoJSON(ctx, api.Request{
			Service: "vm",
			Region:  region,
			Method:  "GET",
			Version: "v1",
			Path:    "/regions/" + region + "/instances",
			Query:   query,
		}, &resp)
		if err != nil {
			return paginate.Page[schema.Host, pageCursor]{}, err
		}

		items := make([]schema.Host, 0, len(resp.Result.Instances))
		for _, i := range resp.Result.Instances {
			ipv4 := i.ElasticIPAddress
			items = append(items, schema.Host{
				HostName:    i.Hostname,
				ID:          i.InstanceID,
				State:       i.Status,
				PublicIPv4:  ipv4,
				PrivateIpv4: i.PrivateIPAddress,
				OSType:      i.OSType,
				Public:      ipv4 != "",
				Region:      region,
			})
		}

		total := resp.Result.TotalCount
		nextSeen := cursor.Seen + len(items)
		done := len(items) == 0
		if total > 0 {
			done = done || nextSeen >= total
		} else {
			done = done || len(items) < vmPageSize
		}
		return paginate.Page[schema.Host, pageCursor]{
			Items: items,
			Next: pageCursor{
				PageNumber: pageNumber + 1,
				Seen:       nextSeen,
			},
			Done: done,
		}, nil
	})
	if err != nil {
		logger.Error("List instances failed.")
		return list, err
	}
	return append(list, got...), nil
}

func (d *Driver) requestRegion() string {
	if d.Region == "" || d.Region == "all" {
		return "cn-north-1"
	}
	return d.Region
}
