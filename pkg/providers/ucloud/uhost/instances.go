package uhost

import (
	"context"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	ucloudauth "github.com/404tk/cloudtoolkit/pkg/providers/ucloud/auth"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

const pageSize = 100

type Driver struct {
	Credential ucloudauth.Credential
	Client     *api.Client
	ProjectID  string
	Regions    []string
}

func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := []schema.Host{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List UHost instances ...")
	}
	if len(d.Regions) == 0 {
		return list, nil
	}

	tracker := processbar.NewRegionTracker()
	defer tracker.Finish()

	got, regionErrs := regionrun.ForEach(ctx, d.Regions, 0, tracker, func(ctx context.Context, region string) ([]schema.Host, error) {
		return d.listRegion(ctx, region)
	})
	list = append(list, got...)
	return list, regionrun.Wrap(regionErrs)
}

func (d *Driver) listRegion(ctx context.Context, region string) ([]schema.Host, error) {
	return paginate.Fetch[schema.Host, int](ctx, func(ctx context.Context, offset int) (paginate.Page[schema.Host, int], error) {
		var resp api.DescribeUHostInstanceResponse
		err := d.client().Do(ctx, api.Request{
			Action: "DescribeUHostInstance",
			Region: region,
			Params: map[string]any{
				"Limit":  pageSize,
				"Offset": offset,
			},
		}, &resp)
		if err != nil {
			return paginate.Page[schema.Host, int]{}, err
		}

		items := make([]schema.Host, 0, len(resp.UHostSet))
		for _, instance := range resp.UHostSet {
			publicIPv4, privateIPv4 := pickIPv4(instance.IPSet)
			items = append(items, schema.Host{
				HostName:    strings.TrimSpace(instance.Name),
				ID:          strings.TrimSpace(instance.UHostID),
				State:       strings.TrimSpace(instance.State),
				PublicIPv4:  publicIPv4,
				PrivateIpv4: privateIPv4,
				OSType:      strings.TrimSpace(instance.OsType),
				Public:      publicIPv4 != "",
				Region:      region,
			})
		}

		next := offset + len(items)
		done := len(items) == 0 || len(items) < pageSize
		if resp.TotalCount > 0 {
			done = next >= resp.TotalCount
		}
		return paginate.Page[schema.Host, int]{
			Items: items,
			Next:  next,
			Done:  done,
		}, nil
	})
}

func (d *Driver) client() *api.Client {
	if d.Client != nil {
		return d.Client
	}
	return api.NewClient(d.Credential, api.WithProjectID(d.ProjectID))
}

func pickIPv4(items []api.UHostIPSet) (string, string) {
	publicIPv4 := ""
	privateIPv4 := ""
	bestWeight := -1

	for _, item := range items {
		ip := strings.TrimSpace(item.IP)
		if ip == "" || strings.EqualFold(strings.TrimSpace(item.IPMode), "IPv6") {
			continue
		}

		if strings.EqualFold(strings.TrimSpace(item.Type), "Private") {
			if privateIPv4 == "" || strings.EqualFold(strings.TrimSpace(item.Default), "true") {
				privateIPv4 = ip
			}
			continue
		}

		if publicIPv4 == "" || item.Weight > bestWeight {
			publicIPv4 = ip
			bestWeight = item.Weight
		}
	}

	return publicIPv4, privateIPv4
}
