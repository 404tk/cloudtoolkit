package iam

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

const iamPageSize = 100

type pageCursor struct {
	PageNumber int
	Seen       int
}

type Driver struct {
	Client *api.Client
}

func (d *Driver) ListUsers(ctx context.Context) ([]schema.User, error) {
	list := []schema.User{}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List IAM users ...")
	}
	if d.Client == nil {
		return list, errors.New("jdcloud iam: nil api client")
	}

	got, err := paginate.Fetch[schema.User, pageCursor](ctx, func(ctx context.Context, cursor pageCursor) (paginate.Page[schema.User, pageCursor], error) {
		pageNumber := cursor.PageNumber
		if pageNumber <= 0 {
			pageNumber = 1
		}

		query := url.Values{}
		query.Set("pageNumber", strconv.Itoa(pageNumber))
		query.Set("pageSize", strconv.Itoa(iamPageSize))

		var resp api.DescribeSubUsersResponse
		err := d.Client.DoJSON(ctx, api.Request{
			Service: "iam",
			// IAM is global. An empty region makes the signer fall back to the
			// jdcloud-api scope expected by this endpoint.
			Region:  "",
			Method:  "GET",
			Version: "v1",
			Path:    "/subUsers",
			Query:   query,
		}, &resp)
		if err != nil {
			return paginate.Page[schema.User, pageCursor]{}, err
		}

		items := make([]schema.User, 0, len(resp.Result.SubUsers))
		for _, user := range resp.Result.SubUsers {
			items = append(items, schema.User{
				UserName:   user.Name,
				UserId:     user.Account,
				CreateTime: user.CreateTime,
			})
		}

		total := resp.Result.Total
		nextSeen := cursor.Seen + len(items)
		done := len(items) == 0
		if total > 0 {
			done = done || nextSeen >= total
		} else {
			done = done || len(items) < iamPageSize
		}
		return paginate.Page[schema.User, pageCursor]{
			Items: items,
			Next: pageCursor{
				PageNumber: pageNumber + 1,
				Seen:       nextSeen,
			},
			Done: done,
		}, nil
	})
	if err != nil {
		logger.Error("List users failed.")
		return list, err
	}
	return append(list, got...), nil
}

func (d *Driver) Validator(_ string) bool {
	if d.Client == nil {
		return false
	}

	query := url.Values{}
	query.Set("pageNumber", "1")
	query.Set("pageSize", "1")

	var resp api.DescribeSubUsersResponse
	err := d.Client.DoJSON(context.Background(), api.Request{
		Service: "iam",
		// IAM is global. An empty region makes the signer fall back to the
		// jdcloud-api scope expected by this endpoint.
		Region:  "",
		Method:  "GET",
		Version: "v1",
		Path:    "/subUsers",
		Query:   query,
	}, &resp)
	if err == nil {
		return true
	}

	var apiErr *api.APIError
	if errors.As(err, &apiErr) && apiErr.IsAuthFailure() {
		return false
	}
	logger.Warning("JDCloud IAM probe inconclusive:", err.Error())
	return true
}
