package udb

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	ucloudauth "github.com/404tk/cloudtoolkit/pkg/providers/ucloud/auth"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

const pageSize = 100

var databaseClassTypes = []struct {
	RequestClass string
	Engine       string
}{
	{RequestClass: "sql", Engine: "MySQL"},
	{RequestClass: "postgresql", Engine: "PostgreSQL"},
	{RequestClass: "nosql", Engine: "MongoDB"},
}

type Driver struct {
	Credential ucloudauth.Credential
	Client     *api.Client
	ProjectID  string
	Regions    []string
}

func (d *Driver) GetDatabases(ctx context.Context) ([]schema.Database, error) {
	list := []schema.Database{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List UDB instances ...")
	}
	if len(d.Regions) == 0 {
		return list, nil
	}

	tracker := processbar.NewRegionTracker()
	defer tracker.Finish()

	got, regionErrs := regionrun.ForEach(ctx, d.Regions, 0, tracker, func(ctx context.Context, region string) ([]schema.Database, error) {
		return d.listRegion(ctx, region)
	})
	list = append(list, got...)
	return list, regionrun.Wrap(regionErrs)
}

func (d *Driver) listRegion(ctx context.Context, region string) ([]schema.Database, error) {
	list := make([]schema.Database, 0)
	errs := make([]string, 0)

	for _, item := range databaseClassTypes {
		databases, err := d.listClassType(ctx, region, item.RequestClass, item.Engine)
		list = append(list, databases...)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", item.RequestClass, err))
		}
	}

	if len(errs) > 0 {
		return list, errors.New(strings.Join(errs, "; "))
	}
	return list, nil
}

func (d *Driver) listClassType(ctx context.Context, region, classType, engine string) ([]schema.Database, error) {
	return paginate.Fetch[schema.Database, int](ctx, func(ctx context.Context, offset int) (paginate.Page[schema.Database, int], error) {
		var resp api.DescribeUDBInstanceResponse
		err := d.client().Do(ctx, api.Request{
			Action: "DescribeUDBInstance",
			Region: region,
			Params: map[string]any{
				"ClassType": classType,
				"Limit":     pageSize,
				"Offset":    offset,
			},
		}, &resp)
		if err != nil {
			return paginate.Page[schema.Database, int]{}, err
		}

		items := make([]schema.Database, 0, len(resp.DataSet))
		for _, instance := range resp.DataSet {
			items = append(items, schema.Database{
				InstanceId:    strings.TrimSpace(instance.DBID),
				Engine:        engine,
				EngineVersion: databaseVersion(instance),
				Region:        region,
				Address:       databaseAddress(instance),
				NetworkType:   databaseNetworkType(instance),
				DBNames:       strings.TrimSpace(instance.Name),
			})
		}

		next := offset + len(items)
		done := len(items) == 0 || len(items) < pageSize
		if resp.TotalCount > 0 {
			done = next >= resp.TotalCount
		}
		return paginate.Page[schema.Database, int]{
			Items: items,
			Next:  next,
			Done:  done,
		}, nil
	})
}

func (d *Driver) client() *api.Client {
	if d.Client != nil {
		return d.Client
	}
	return api.NewClient(d.Credential, api.WithProjectID(d.ProjectID))
}

func databaseVersion(instance api.UDBInstanceSet) string {
	if version := strings.TrimSpace(instance.DBSubVersion); version != "" {
		return version
	}
	return strings.TrimSpace(instance.DBTypeID)
}

func databaseAddress(instance api.UDBInstanceSet) string {
	host := strings.TrimSpace(instance.VirtualIP)
	if host == "" {
		return ""
	}
	if instance.Port <= 0 {
		return host
	}
	return fmt.Sprintf("%s:%d", host, instance.Port)
}

func databaseNetworkType(instance api.UDBInstanceSet) string {
	if strings.TrimSpace(instance.VPCID) != "" {
		return "VPC"
	}
	if strings.TrimSpace(instance.SubnetID) != "" {
		return "Private"
	}
	return ""
}
