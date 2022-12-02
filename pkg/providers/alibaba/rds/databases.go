package rds

import (
	"context"
	"log"
	"math"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/rds"
)

type RdsProvider struct {
	Client         *rds.Client
	ResourceGroups []string
}

func (d *RdsProvider) GetDatabases(ctx context.Context) ([]*schema.Database, error) {
	list := schema.NewResources().Databases
	log.Println("[*] Start enumerating RDS ...")
	for _, resourceGroupId := range d.ResourceGroups {
		page := 1
		for {
			describeDBInstancesRequest := rds.CreateDescribeDBInstancesRequest()
			if resourceGroupId != "" {
				describeDBInstancesRequest.ResourceGroupId = resourceGroupId
			}
			describeDBInstancesRequest.PageSize = requests.NewInteger(100)
			describeDBInstancesRequest.PageNumber = requests.NewInteger(page)
			response, err := d.Client.DescribeDBInstances(describeDBInstancesRequest)
			if err != nil {
				log.Println("[-] Enumerate RDS failed.")
				return list, err
			}
			pageCount := int(math.Ceil(float64(response.TotalRecordCount) / 100))
			for _, dbInstance := range response.Items.DBInstance {

				_dbInstance := &schema.Database{
					DBInstanceId:  dbInstance.DBInstanceId,
					Engine:        dbInstance.Engine,
					EngineVersion: dbInstance.EngineVersion,
					Region:        dbInstance.RegionId,
				}

				list = append(list, _dbInstance)
			}
			if page == pageCount || pageCount == 0 {
				break
			}
			page++
		}
	}
	return list, nil
}
