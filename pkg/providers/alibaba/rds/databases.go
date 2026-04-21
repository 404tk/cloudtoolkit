package rds

import (
	"context"
	"strings"
	"sync"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	aliauth "github.com/404tk/cloudtoolkit/pkg/providers/alibaba/auth"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

type Driver struct {
	Cred          aliauth.Credential
	Region        string
	clientOptions []api.Option
	partialErr    error
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

func (d *Driver) newClient() *api.Client {
	return api.NewClient(d.Cred, d.clientOptions...)
}

func (d *Driver) SetClientOptions(opts ...api.Option) {
	d.clientOptions = append([]api.Option(nil), opts...)
}

func (d *Driver) GetDatabases(ctx context.Context) ([]schema.Database, error) {
	list := []schema.Database{}
	d.partialErr = nil
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List RDS instances ...")
	}
	defer func() { SetCacheDBList(list) }()
	client := d.newClient()
	regions := []string{api.NormalizeRegion(d.Region)}
	if d.Region == "all" {
		var err error
		regions, err = describeRegions(ctx, client)
		if err != nil {
			logger.Error("Describe regions failed.")
			return list, err
		}
		if len(regions) == 0 {
			return list, nil
		}
	}
	tracker := processbar.NewRegionTracker()
	defer tracker.Finish()
	got, regionErrs := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, r string) ([]schema.Database, error) {
		return paginate.Fetch(ctx, func(ctx context.Context, page int) (paginate.Page[schema.Database, int], error) {
			if page == 0 {
				page = 1
			}
			response, err := client.DescribeRDSInstances(ctx, r, page, 100)
			if err != nil {
				return paginate.Page[schema.Database, int]{}, err
			}
			items := make([]schema.Database, 0, len(response.Items.DBInstance))
			for _, dbInstance := range response.Items.DBInstance {
				items = append(items, schema.Database{
					InstanceId:    dbInstance.DBInstanceID,
					Engine:        dbInstance.Engine,
					EngineVersion: dbInstance.EngineVersion,
					Region:        dbInstance.RegionID,
					Address:       dbInstance.ConnectionString,
					NetworkType:   dbInstance.InstanceNetworkType,
					DBNames:       describeDatabases(ctx, client, r, dbInstance.DBInstanceID),
				})
			}
			return paginate.Page[schema.Database, int]{
				Items: items,
				Next:  page + 1,
				Done:  isLastPage(page, response.PageRecordCount, response.TotalRecordCount, len(response.Items.DBInstance)),
			}, nil
		})
	})
	list = append(list, got...)
	d.partialErr = regionrun.Wrap(regionErrs)
	return list, nil
}

func (d *Driver) PartialError() error {
	return d.partialErr
}

func describeRegions(ctx context.Context, client *api.Client) ([]string, error) {
	response, err := client.DescribeRDSRegions(ctx, api.DefaultRegion)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(response.Regions.RDSRegion))
	regions := make([]string, 0, len(response.Regions.RDSRegion))
	for _, region := range response.Regions.RDSRegion {
		if _, ok := seen[region.RegionID]; ok {
			continue
		}
		seen[region.RegionID] = struct{}{}
		regions = append(regions, region.RegionID)
	}
	return regions, nil
}

func describeDatabases(ctx context.Context, client *api.Client, region, instanceID string) string {
	response, err := client.DescribeRDSDatabases(ctx, region, instanceID)
	if err != nil {
		logger.Error(err)
		return ""
	}
	dbs := make([]string, 0, len(response.Databases.Database))
	for _, db := range response.Databases.Database {
		dbs = append(dbs, db.DBName)
	}
	return strings.Join(dbs, ",")
}

func isLastPage(page, pageSize, totalCount, items int) bool {
	if items == 0 {
		return true
	}
	if pageSize <= 0 {
		pageSize = items
	}
	if totalCount <= 0 {
		return items < pageSize
	}
	return page*pageSize >= totalCount
}
