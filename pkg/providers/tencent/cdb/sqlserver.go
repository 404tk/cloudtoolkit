package cdb

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	sqlserver "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sqlserver/v20180328"
)

func (d *Driver) ListSQLServer(ctx context.Context) ([]schema.Database, error) {
	list := []schema.Database{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List SQLServer ...")
	}
	cpf := profile.NewClientProfile()
	var regions []string
	if d.Region == "all" {
		client, err := sqlserver.NewClient(d.Credential, "ap-guangzhou", cpf)
		if err != nil {
			return list, err
		}
		req := sqlserver.NewDescribeRegionsRequest()
		resp, err := client.DescribeRegions(req)
		if err != nil {
			logger.Error("list regions failed.")
			return list, err
		}
		for _, r := range resp.Response.RegionSet {
			if *r.RegionState == "AVAILABLE" {
				regions = append(regions, *r.Region)
			}
		}
	} else {
		regions = append(regions, d.Region)
	}

	tracker := processbar.NewRegionTracker()
	defer tracker.Finish()
	got, _ := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, r string) ([]schema.Database, error) {
		var regionList []schema.Database
		client, err := sqlserver.NewClient(d.Credential, r, cpf)
		if err != nil {
			return regionList, err
		}
		request := sqlserver.NewDescribeDBInstancesRequest()
		response, err := client.DescribeDBInstances(request)
		if err != nil {
			return regionList, err
		}
		for _, instance := range response.Response.DBInstances {
			_db := schema.Database{
				InstanceId:    *instance.InstanceId,
				Engine:        *instance.VersionName,
				EngineVersion: *instance.Version,
				Region:        *instance.Region,
			}
			if *instance.DnsPodDomain != "" {
				_db.Address = fmt.Sprintf("%s:%d", *instance.DnsPodDomain, *instance.TgwWanVPort)
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
