package rds

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/volcengine/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
)

const pageSize int32 = 100

type Driver struct {
	Client     *api.Client
	Region     string
	partialErr error
}

var errNilAPIClient = errors.New("volcengine rds: nil api client")

func (d *Driver) PartialError() error {
	return d.partialErr
}

func (d *Driver) ListMySQL(ctx context.Context) ([]schema.Database, error) {
	return d.listInstances(ctx, "List RDS MySQL instances ...", api.ServiceRDSMySQL, func(ctx context.Context, region string, pageNumber int32) ([]schema.Database, int32, error) {
		resp, err := d.Client.DescribeRDSMySQLInstances(ctx, region, pageNumber, pageSize)
		if err != nil {
			return nil, 0, err
		}
		items := make([]schema.Database, 0, len(resp.Result.Instances))
		for _, instance := range resp.Result.Instances {
			address, networkType := pickAddress(instance.AddressObject)
			instanceID := strings.TrimSpace(instance.InstanceID)
			engineVersion := strings.TrimSpace(instance.DBEngineVersion)
			regionID := strings.TrimSpace(instance.RegionID)
			items = append(items, schema.Database{
				InstanceId:    instanceID,
				Engine:        "MySQL",
				EngineVersion: normalizeVersion(engineVersion, "MySQL_"),
				Region:        regionID,
				Address:       address,
				NetworkType:   networkType,
			})
		}
		return items, resp.Result.Total, nil
	})
}

func (d *Driver) ListPostgreSQL(ctx context.Context) ([]schema.Database, error) {
	return d.listInstances(ctx, "List RDS PostgreSQL instances ...", api.ServiceRDSPostgreSQL, func(ctx context.Context, region string, pageNumber int32) ([]schema.Database, int32, error) {
		resp, err := d.Client.DescribeRDSPostgreSQLInstances(ctx, region, pageNumber, pageSize)
		if err != nil {
			return nil, 0, err
		}
		items := make([]schema.Database, 0, len(resp.Result.Instances))
		for _, instance := range resp.Result.Instances {
			address, networkType := pickAddress(instance.AddressObject)
			instanceID := strings.TrimSpace(instance.InstanceID)
			engineVersion := strings.TrimSpace(instance.DBEngineVersion)
			regionID := strings.TrimSpace(instance.RegionID)
			items = append(items, schema.Database{
				InstanceId:    instanceID,
				Engine:        "PostgreSQL",
				EngineVersion: normalizeVersion(engineVersion, "PostgreSQL_"),
				Region:        regionID,
				Address:       address,
				NetworkType:   networkType,
			})
		}
		return items, resp.Result.Total, nil
	})
}

func (d *Driver) ListSQLServer(ctx context.Context) ([]schema.Database, error) {
	return d.listInstances(ctx, "List RDS SQL Server instances ...", api.ServiceRDSMSSQL, func(ctx context.Context, region string, pageNumber int32) ([]schema.Database, int32, error) {
		resp, err := d.Client.DescribeRDSSQLServerInstances(ctx, region, pageNumber, pageSize)
		if err != nil {
			return nil, 0, err
		}
		items := make([]schema.Database, 0, len(resp.Result.InstancesInfo))
		for _, instance := range resp.Result.InstancesInfo {
			instanceID := strings.TrimSpace(instance.InstanceID)
			engineVersion := strings.TrimSpace(instance.DBEngineVersion)
			regionID := strings.TrimSpace(instance.RegionID)
			items = append(items, schema.Database{
				InstanceId:    instanceID,
				Engine:        "SQL Server",
				EngineVersion: engineVersion,
				Region:        regionID,
				Address:       pickSQLServerAddress(instance.NodeDetailInfo, instance.Port),
			})
		}
		return items, resp.Result.Total, nil
	})
}

func (d *Driver) listInstances(
	ctx context.Context,
	logMessage string,
	service string,
	fetch func(ctx context.Context, region string, pageNumber int32) ([]schema.Database, int32, error),
) ([]schema.Database, error) {
	list := []schema.Database{}
	d.partialErr = nil
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info(logMessage)
	}
	if d.Client == nil {
		return list, errNilAPIClient
	}

	regions, err := d.getRegions(ctx, service)
	if err != nil {
		logger.Error("List regions failed.")
		return list, err
	}
	if len(regions) == 0 {
		return list, nil
	}

	seedErrs := map[string]error{}
	tracker := processbar.NewRegionTracker()
	trackerUsed := false
	defer func() {
		if trackerUsed {
			tracker.Finish()
		}
	}()
	if d.Region == "all" && len(regions) > 0 {
		probeRegion := regions[0]
		probeItems, probeErr := d.listRegion(ctx, probeRegion, fetch)
		if probeErr != nil {
			if api.IsAccessDenied(probeErr) {
				return list, probeErr
			}
			seedErrs[probeRegion] = probeErr
			tracker.Update(probeRegion, 0)
			trackerUsed = true
		} else {
			list = append(list, probeItems...)
			tracker.Update(probeRegion, len(probeItems))
			trackerUsed = true
		}
		regions = regions[1:]
	}
	if len(regions) == 0 {
		d.partialErr = regionrun.Wrap(seedErrs)
		return list, nil
	}

	trackerUsed = true
	got, regionErrs := regionrun.ForEach(ctx, regions, 0, tracker, func(ctx context.Context, region string) ([]schema.Database, error) {
		return d.listRegion(ctx, region, fetch)
	})
	list = append(list, got...)
	d.partialErr = regionrun.Wrap(mergeRegionErrors(seedErrs, regionErrs))
	return list, nil
}

func (d *Driver) getRegions(ctx context.Context, service string) ([]string, error) {
	if d.Region != "all" {
		return []string{d.requestRegion()}, nil
	}
	resp, err := d.Client.DescribeRDSRegions(ctx, service, d.requestRegion())
	if err != nil {
		return nil, err
	}
	regions := make([]string, 0, len(resp.Result.Regions))
	seen := make(map[string]struct{}, len(resp.Result.Regions))
	for _, region := range resp.Result.Regions {
		regionID := strings.TrimSpace(region.RegionID)
		if regionID == "" {
			continue
		}
		if _, ok := seen[regionID]; ok {
			continue
		}
		seen[regionID] = struct{}{}
		regions = append(regions, regionID)
	}
	return regions, nil
}

func (d *Driver) requestRegion() string {
	region := strings.TrimSpace(d.Region)
	if region == "" || region == "all" {
		return api.DefaultRegion
	}
	return region
}

func (d *Driver) listRegion(
	ctx context.Context,
	region string,
	fetch func(ctx context.Context, region string, pageNumber int32) ([]schema.Database, int32, error),
) ([]schema.Database, error) {
	return paginate.Fetch[schema.Database, int32](ctx, func(ctx context.Context, pageNumber int32) (paginate.Page[schema.Database, int32], error) {
		if pageNumber == 0 {
			pageNumber = 1
		}
		items, total, err := fetch(ctx, region, pageNumber)
		if err != nil {
			return paginate.Page[schema.Database, int32]{}, err
		}
		done := len(items) == 0 || int32(len(items)) < pageSize
		if total > 0 {
			done = pageNumber*pageSize >= total
		}
		return paginate.Page[schema.Database, int32]{
			Items: items,
			Next:  pageNumber + 1,
			Done:  done,
		}, nil
	})
}

func mergeRegionErrors(base, extra map[string]error) map[string]error {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	merged := make(map[string]error, len(base)+len(extra))
	for region, err := range base {
		if err != nil {
			merged[region] = err
		}
	}
	for region, err := range extra {
		if err != nil {
			merged[region] = err
		}
	}
	return merged
}

func pickAddress(addresses []api.RDSAddressObject) (string, string) {
	bestAddress := ""
	bestNetworkType := ""
	for _, item := range addresses {
		domain := strings.TrimSpace(item.Domain)
		ipAddress := strings.TrimSpace(item.IPAddress)
		host := firstNonEmpty(domain, ipAddress)
		if host == "" {
			continue
		}
		port := strings.TrimSpace(item.Port)
		networkType := strings.TrimSpace(item.NetworkType)
		address := formatAddress(host, port)
		if strings.EqualFold(networkType, "Public") {
			return address, networkType
		}
		if bestAddress == "" {
			bestAddress = address
			bestNetworkType = networkType
		}
	}
	return bestAddress, bestNetworkType
}

func pickSQLServerAddress(nodes []api.RDSSQLServerNode, port string) string {
	port = strings.TrimSpace(port)
	for _, node := range nodes {
		nodeType := strings.TrimSpace(node.NodeType)
		if !strings.EqualFold(nodeType, "Primary") {
			continue
		}
		host := strings.TrimSpace(node.NodeIP)
		if host != "" {
			return formatAddress(host, port)
		}
	}
	for _, node := range nodes {
		host := strings.TrimSpace(node.NodeIP)
		if host != "" {
			return formatAddress(host, port)
		}
	}
	return ""
}

func formatAddress(host, port string) string {
	host = strings.TrimSpace(host)
	port = strings.TrimSpace(port)
	switch {
	case host == "":
		return ""
	case port == "":
		return host
	default:
		return fmt.Sprintf("%s:%s", host, port)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if v := strings.TrimSpace(value); v != "" {
			return v
		}
	}
	return ""
}

func normalizeVersion(value, prefix string) string {
	value = strings.TrimSpace(value)
	if prefix == "" || !strings.HasPrefix(value, prefix) {
		return value
	}
	version := strings.TrimPrefix(value, prefix)
	version = strings.ReplaceAll(version, "_", ".")
	return strings.Trim(version, ".")
}
