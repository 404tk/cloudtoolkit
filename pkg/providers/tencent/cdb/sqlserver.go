package cdb

import (
	"context"
	"fmt"

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
		logger.Info("Start enumerating SQLServer ...")
	}
	cpf := profile.NewClientProfile()
	var regions []string
	if d.Region == "all" {
		client, _ := sqlserver.NewClient(d.Credential, "ap-guangzhou", cpf)
		req := sqlserver.NewDescribeRegionsRequest()
		resp, err := client.DescribeRegions(req)
		if err != nil {
			logger.Error("Enumerate SQLServer failed.")
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

	flag := false
	prevLength := 0
	for _, r := range regions {
		client, _ := sqlserver.NewClient(d.Credential, r, cpf)
		request := sqlserver.NewDescribeDBInstancesRequest()
		response, err := client.DescribeDBInstances(request)
		if err != nil {
			logger.Error("Enumerate SQLServer failed.")
			return list, err
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
			list = append(list, _db)
		}
		select {
		case <-ctx.Done():
			goto done
		default:
			prevLength, flag = processbar.RegionPrint(r, len(response.Response.DBInstances), prevLength, flag)
		}
	}
done:
	if !flag {
		fmt.Printf("\n\033[F\033[K")
	}
	return list, nil
}
