package cdb

import (
	"context"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
	cdb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cdb/v20170320"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

type Driver struct {
	Credential *common.Credential
	Region     string
}

func (d *Driver) ListMySQL(ctx context.Context) ([]schema.Database, error) {
	list := []schema.Database{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List MySQL ...")
	}
	cpf := profile.NewClientProfile()
	var regions []string
	if d.Region == "all" {
		client, err := cdb.NewClient(d.Credential, "ap-guangzhou", cpf)
		if err != nil {
			return list, err
		}
		req := cdb.NewDescribeCdbZoneConfigRequest()
		resp, err := client.DescribeCdbZoneConfig(req)
		if err != nil {
			logger.Error("List regions failed.")
			return list, err
		}
		for _, r := range resp.Response.DataResult.Regions {
			regions = append(regions, *r.Region)
		}
	} else {
		regions = append(regions, d.Region)
	}

	tracker := processbar.NewRegionTracker()
	defer tracker.Finish()
	got, _ := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, r string) ([]schema.Database, error) {
		var regionList []schema.Database
		client, err := cdb.NewClient(d.Credential, r, cpf)
		if err != nil {
			return regionList, err
		}
		request := cdb.NewDescribeDBInstancesRequest()
		response, err := client.DescribeDBInstances(request)
		if err != nil {
			if strings.Contains(err.Error(), "UnsupportedRegion") {
				return regionList, nil
			}
			return regionList, err
		}
		for _, instance := range response.Response.Items {
			_db := schema.Database{
				InstanceId:    *instance.InstanceId,
				Engine:        "MySQL",
				EngineVersion: *instance.EngineVersion,
				Region:        *instance.Region,
			}
			if *instance.WanStatus == 1 {
				_db.Address = fmt.Sprintf("%s:%d", *instance.WanDomain, *instance.WanPort)
			} else {
				_db.Address = fmt.Sprintf("%s:%d", *instance.Vip, *instance.Vport)
			}
			regionList = append(regionList, _db)
		}
		return regionList, nil
	})
	list = append(list, got...)
	return list, nil
}
