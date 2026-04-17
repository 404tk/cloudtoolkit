package cvm

import (
	"context"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

type Driver struct {
	Credential *common.Credential
	Region     string
}

func (d *Driver) NewClient() (*cvm.Client, error) {
	cpf := profile.NewClientProfile()
	region := d.Region
	if region == "all" || region == "" {
		region = "ap-guangzhou"
	}
	return cvm.NewClient(d.Credential, region, cpf)
}

func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := []schema.Host{}
	logger.Info("List CVM instances ...")
	var regions []string
	if d.Region == "all" {
		client, err := d.NewClient()
		if err != nil {
			return list, err
		}
		req := cvm.NewDescribeRegionsRequest()
		resp, err := client.DescribeRegions(req)
		if err != nil {
			logger.Error("List regions failed.")
			return list, err
		}
		for _, r := range resp.Response.RegionSet {
			regions = append(regions, *r.Region)
		}
	} else {
		regions = append(regions, d.Region)
	}
	tracker := processbar.NewRegionTracker()
	defer tracker.Finish()
	got, _ := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, r string) ([]schema.Host, error) {
		var regionList []schema.Host
		client, err := cvm.NewClient(d.Credential, r, profile.NewClientProfile())
		if err != nil {
			return regionList, err
		}
		request := cvm.NewDescribeInstancesRequest()
		request.Limit = common.Int64Ptr(100)
		var offset int64 = 0
		for {
			request.Offset = common.Int64Ptr(offset)
			response, err := client.DescribeInstances(request)
			if err != nil {
				return regionList, err
			}
			for _, instance := range response.Response.InstanceSet {
				var ipv4, privateIPv4 string
				if len(instance.PublicIpAddresses) > 0 {
					ipv4 = *instance.PublicIpAddresses[0]
				}
				if len(instance.PrivateIpAddresses) > 0 {
					privateIPv4 = *instance.PrivateIpAddresses[0]
				}
				host := schema.Host{
					HostName:    *instance.InstanceName,
					ID:          *instance.InstanceId,
					State:       *instance.InstanceState,
					PublicIPv4:  ipv4,
					PrivateIpv4: privateIPv4,
					Public:      ipv4 != "",
					Region:      r,
				}
				os_name := strings.Split(*instance.OsName, " ")[0]
				if os_name == "Windows" {
					host.OSType = "WINDOWS"
				} else {
					host.OSType = "LINUX_UNIX"
				}
				regionList = append(regionList, host)
			}
			if len(response.Response.InstanceSet) < 100 {
				break
			}
			offset += 100
			select {
			case <-ctx.Done():
				return regionList, nil
			default:
			}
		}
		return regionList, nil
	})
	list = append(list, got...)

	return list, nil
}
