package cvm

import (
	"context"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/api"
	"github.com/404tk/cloudtoolkit/pkg/providers/tencent/auth"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

type Driver struct {
	Credential    auth.Credential
	Region        string
	clientOptions []api.Option
}

func (d *Driver) newClient() *api.Client {
	return api.NewClient(d.Credential, d.clientOptions...)
}

func (d *Driver) SetClientOptions(opts ...api.Option) {
	d.clientOptions = append([]api.Option(nil), opts...)
}

func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := []schema.Host{}
	logger.Info("List CVM instances ...")
	var regions []string
	client := d.newClient()
	if d.Region == "all" {
		resp, err := client.DescribeCVMRegions(ctx, d.Region)
		if err != nil {
			logger.Error("List regions failed.")
			return list, err
		}
		for _, r := range resp.Response.RegionSet {
			region := derefString(r.Region)
			if region == "" {
				continue
			}
			regions = append(regions, region)
		}
	} else {
		regions = append(regions, normalizedRegion(d.Region))
	}
	tracker := processbar.NewRegionTracker()
	defer tracker.Finish()
	got, _ := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, r string) ([]schema.Host, error) {
		return paginate.Fetch(ctx, func(ctx context.Context, offset int64) (paginate.Page[schema.Host, int64], error) {
			response, err := client.DescribeCVMInstances(ctx, r, offset, 100)
			if err != nil {
				return paginate.Page[schema.Host, int64]{}, err
			}
			items := make([]schema.Host, 0, len(response.Response.InstanceSet))
			for _, instance := range response.Response.InstanceSet {
				items = append(items, mapHost(instance, r))
			}
			return paginate.Page[schema.Host, int64]{
				Items: items,
				Next:  offset + int64(len(response.Response.InstanceSet)),
				Done:  doneByTotal(offset, int64(len(response.Response.InstanceSet)), derefInt64(response.Response.TotalCount), 100),
			}, nil
		})
	})
	list = append(list, got...)

	return list, nil
}

func mapHost(instance api.CVMInstanceInfo, region string) schema.Host {
	ipv4 := firstString(instance.PublicIPAddresses)
	privateIPv4 := firstString(instance.PrivateIPAddresses)
	host := schema.Host{
		HostName:    derefString(instance.InstanceName),
		ID:          derefString(instance.InstanceID),
		State:       derefString(instance.InstanceState),
		PublicIPv4:  ipv4,
		PrivateIpv4: privateIPv4,
		Public:      ipv4 != "",
		Region:      region,
	}
	if strings.EqualFold(strings.Split(derefString(instance.OSName), " ")[0], "Windows") {
		host.OSType = "WINDOWS"
	} else {
		host.OSType = "LINUX_UNIX"
	}
	return host
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func derefInt64(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

func doneByTotal(offset, count, total, limit int64) bool {
	if count == 0 {
		return true
	}
	if total <= 0 {
		return count < limit
	}
	return offset+count >= total
}

func normalizedRegion(region string) string {
	switch region {
	case "", "all":
		return api.DefaultRegion
	default:
		return region
	}
}
