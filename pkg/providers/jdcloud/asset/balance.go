package asset

import (
	"context"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Client *api.Client
	Region string
}

func (d *Driver) QueryAccountBalance(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
	}
	if d.Client == nil {
		return
	}

	region := d.requestRegion()
	var resp api.DescribeAccountAmountResponse
	err := d.Client.DoJSON(ctx, api.Request{
		Service: "asset",
		Region:  region,
		Method:  "GET",
		Version: "v1",
		Path:    "/regions/" + region + "/assets:describeAccountAmount",
	}, &resp)
	if err == nil && resp.Result.AvailableAmount != "" {
		logger.Warning("Available cash amount:", resp.Result.AvailableAmount)
	}
}

func (d *Driver) requestRegion() string {
	region := strings.TrimSpace(d.Region)
	switch {
	case region == "", strings.EqualFold(region, "all"):
		return "cn-north-1"
	default:
		return region
	}
}
