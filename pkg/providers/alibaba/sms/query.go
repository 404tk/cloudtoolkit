package sms

import (
	"context"
	"time"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
)

func (d *Driver) querySendStatistics(ctx context.Context, client *api.Client, region string) (int64, error) {
	now := time.Now
	if d.now != nil {
		now = d.now
	}
	date := now().UTC().Format("20060102")
	response, err := client.QuerySMSSendStatistics(ctx, region, date)
	if err != nil {
		return 0, err
	}
	return response.Data.TotalSize, nil
}
