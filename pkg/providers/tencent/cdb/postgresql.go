package cdb

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	postgres "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/postgres/v20170312"
)

func (d *Driver) ListPostgreSQL(ctx context.Context) ([]schema.Database, error) {
	list := []schema.Database{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List PostgreSQL ...")
	}
	cpf := profile.NewClientProfile()
	var regions []string
	if d.Region == "all" {
		client, err := postgres.NewClient(d.Credential, "ap-guangzhou", cpf)
		if err != nil {
			return list, err
		}
		req := postgres.NewDescribeRegionsRequest()
		resp, err := client.DescribeRegions(req)
		if err != nil {
			logger.Error("List regions failed:", err)
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
		client, err := postgres.NewClient(d.Credential, r, cpf)
		if err != nil {
			continue
		}
		request := postgres.NewDescribeDBInstancesRequest()
		response, err := client.DescribeDBInstances(request)
		if err != nil {
			fmt.Println()
			logger.Error("DescribeDBInstances failed:", err)
			return list, err
		}

		for _, instance := range response.Response.DBInstanceSet {
			_db := schema.Database{
				InstanceId:    *instance.DBInstanceId,
				Engine:        *instance.DBEngine,
				EngineVersion: *instance.DBInstanceVersion,
				Region:        *instance.Region,
			}
			for _, info := range instance.DBInstanceNetInfo {
				if *info.NetType == "public" && *info.Status == "opened" {
					_db.Address = fmt.Sprintf("%s:%d", *info.Address, *info.Port)
					break
				} else {
					_db.Address = fmt.Sprintf("%s:%d", *info.Ip, *info.Port)
				}
			}
			list = append(list, _db)
		}
		select {
		case <-ctx.Done():
			goto done
		default:
			prevLength, flag = processbar.RegionPrint(r, len(response.Response.DBInstanceSet), prevLength, flag)
		}
	}
done:
	if !flag {
		fmt.Printf("\n\033[F\033[K")
	}
	return list, nil
}
