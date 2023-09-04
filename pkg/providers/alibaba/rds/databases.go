package rds

import (
	"context"
	"math"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/rds"
)

type Driver struct {
	Cred   *credentials.StsTokenCredential
	Region string
}

func (d *Driver) GetDatabases(ctx context.Context) ([]schema.Database, error) {
	list := schema.NewResources().Databases
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("Start enumerating RDS ...")
	}
	region := d.Region
	if region == "all" {
		region = "cn-hangzhou"
	}
	client, err := rds.NewClientWithOptions(region, sdk.NewConfig(), d.Cred)
	if err != nil {
		return list, err
	}
	page := 1
	for {
		describeDBInstancesRequest := rds.CreateDescribeDBInstancesRequest()
		describeDBInstancesRequest.PageSize = requests.NewInteger(100)
		describeDBInstancesRequest.PageNumber = requests.NewInteger(page)
		response, err := client.DescribeDBInstances(describeDBInstancesRequest)
		if err != nil {
			logger.Error("Describe database instances failed.")
			return list, err
		}
		pageCount := int(math.Ceil(float64(response.TotalRecordCount) / 100))
		for _, dbInstance := range response.Items.DBInstance {
			_db := schema.Database{
				DBInstanceId:  dbInstance.DBInstanceId,
				Engine:        dbInstance.Engine,
				EngineVersion: dbInstance.EngineVersion,
				Region:        dbInstance.RegionId,
			}
			if dbInstance.DBInstanceNetType == "Internet" {
				_db.Address = dbInstance.ConnectionString
			}

			list = append(list, _db)
		}
		if page == pageCount || pageCount == 0 {
			break
		}
		page++
		select {
		case <-ctx.Done():
			return list, nil
		default:
			continue
		}
	}
	return list, nil
}
