package rds

import (
	"context"
	"math"
	"strings"

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

var CacheDBList []schema.Database

func (d *Driver) NewClient() (*rds.Client, error) {
	region := d.Region
	if region == "all" {
		region = "cn-hangzhou"
	}
	return rds.NewClientWithOptions(region, sdk.NewConfig(), d.Cred)
}

func (d *Driver) GetDatabases(ctx context.Context) ([]schema.Database, error) {
	list := schema.NewResources().Databases
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("Start enumerating RDS ...")
	}
	defer func() { CacheDBList = list }()
	client, err := d.NewClient()
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
				InstanceId:    dbInstance.DBInstanceId,
				Engine:        dbInstance.Engine,
				EngineVersion: dbInstance.EngineVersion,
				Region:        dbInstance.RegionId,
				Address:       dbInstance.ConnectionString,
				NetworkType:   dbInstance.InstanceNetworkType,
			}
			// if dbInstance.DBInstanceNetType == "Internet" {
			// _db.Address = dbInstance.ConnectionString
			// }
			_db.DBNames = describeDatabases(client, dbInstance.DBInstanceId)
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

func describeDatabases(client *rds.Client, instanceId string) string {
	request := rds.CreateDescribeDatabasesRequest()
	request.Scheme = "https"
	request.DBInstanceId = instanceId
	request.PageSize = requests.NewInteger(30)
	request.PageNumber = requests.NewInteger(1)
	request.DBStatus = "Running"
	response, err := client.DescribeDatabases(request)
	if err != nil {
		logger.Error(err)
		return ""
	}
	dbs := []string{}
	for _, db := range response.Databases.Database {
		dbs = append(dbs, db.DBName)
	}
	return strings.Join(dbs, ",")
}
