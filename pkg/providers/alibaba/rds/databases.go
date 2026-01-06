package rds

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/rds"
)

type Driver struct {
	Cred   *credentials.StsTokenCredential
	Region string
}

var (
	CacheDBList []schema.Database
	dbCacheMu   sync.RWMutex
)

func SetCacheDBList(dbs []schema.Database) {
	dbCacheMu.Lock()
	defer dbCacheMu.Unlock()
	CacheDBList = dbs
}

func GetCacheDBList() []schema.Database {
	dbCacheMu.RLock()
	defer dbCacheMu.RUnlock()
	return CacheDBList
}

func (d *Driver) NewClient() (*rds.Client, error) {
	region := d.Region
	if region == "all" {
		region = "cn-hangzhou"
	}
	return rds.NewClientWithOptions(region, sdk.NewConfig(), d.Cred)
}

func (d *Driver) GetDatabases(ctx context.Context) ([]schema.Database, error) {
	list := []schema.Database{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List RDS instances ...")
	}
	defer func() { SetCacheDBList(list) }()
	client, err := d.NewClient()
	if err != nil {
		return list, err
	}
	var regions = []string{d.Region}
	if d.Region == "all" {
		regions = describeRegions(client)
		if len(regions) == 0 {
			return list, nil
		}
	}
	flag := false
	prevLength := 0
	count := 0
	for _, r := range regions {
		page := 1
		for {
			request := rds.CreateDescribeDBInstancesRequest()
			request.PageSize = requests.NewInteger(100)
			request.PageNumber = requests.NewInteger(page)
			if r != "all" {
				request.RegionId = r
			}

			response, err := client.DescribeDBInstances(request)
			if err != nil {
				break
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
		}
		select {
		case <-ctx.Done():
			goto done
		default:
			prevLength, flag = processbar.RegionPrint(r, len(list)-count, prevLength, flag)
			count = len(list)
		}
	}
done:
	if !flag {
		fmt.Printf("\n\033[F\033[K")
	}
	return list, nil
}

func describeRegions(client *rds.Client) []string {
	request := rds.CreateDescribeRegionsRequest()
	request.Scheme = "https"
	request.AcceptLanguage = "en-US"
	response, err := client.DescribeRegions(request)
	if err != nil {
		logger.Error(err)
		return []string{}
	}
	temp := make(map[string]string)
	for _, region := range response.Regions.RDSRegion {
		temp[region.RegionId] = ""
	}
	regions := []string{}
	for region := range temp {
		regions = append(regions, region)
	}
	return regions
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
