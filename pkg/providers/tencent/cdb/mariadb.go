package cdb

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	mariadb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/mariadb/v20170312"
)

func (d *Driver) ListMariaDB(ctx context.Context) ([]schema.Database, error) {
	list := []schema.Database{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List MariaDB ...")
	}
	cpf := profile.NewClientProfile()
	var regions []string
	if d.Region == "all" {
		client, err := mariadb.NewClient(d.Credential, "ap-guangzhou", cpf)
		if err != nil {
			return list, err
		}
		req := mariadb.NewDescribeSaleInfoRequest()
		resp, err := client.DescribeSaleInfo(req)
		if err != nil {
			logger.Error("List regions failed.")
			return list, err
		}
		for _, r := range resp.Response.RegionList {
			regions = append(regions, *r.Region)
		}
	} else {
		regions = append(regions, d.Region)
	}

	tracker := processbar.NewRegionTracker()
	defer tracker.Finish()
	got, _ := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, r string) ([]schema.Database, error) {
		var regionList []schema.Database
		client, err := mariadb.NewClient(d.Credential, r, cpf)
		if err != nil {
			return regionList, err
		}
		request := mariadb.NewDescribeDBInstancesRequest()
		response, err := client.DescribeDBInstances(request)
		if err != nil {
			return regionList, err
		}
		for _, instance := range response.Response.Instances {
			_db := schema.Database{
				InstanceId:    *instance.InstanceId,
				Engine:        "MariaDB",
				EngineVersion: *instance.DbVersion,
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
