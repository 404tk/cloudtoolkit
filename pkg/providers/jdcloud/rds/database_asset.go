package rds

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

const (
	defaultDescribeRegion = "cn-north-1"
	listPageSize          = 100
	listMaxPages          = 50
)

// GetDatabases lists JDCloud RDS instances in the configured region and
// surfaces them as the cloudlist `database` asset.
func (d *Driver) GetDatabases(ctx context.Context) ([]schema.Database, error) {
	out := []schema.Database{}
	if d == nil || d.Client == nil {
		return out, errors.New("jdcloud rds: nil api client")
	}
	logger.Info("List JDCloud RDS instances ...")
	region := strings.TrimSpace(d.Region)
	if region == "" || region == "all" {
		region = defaultDescribeRegion
	}
	for page := 1; page <= listMaxPages; page++ {
		resp, err := d.Client.DescribeRDSInstances(ctx, region, page, listPageSize)
		if err != nil {
			return out, err
		}
		for _, inst := range resp.Result.DBInstances {
			network := "Private"
			address := inst.InternalDomain
			if inst.InternalPort > 0 && inst.InternalDomain != "" {
				address = fmt.Sprintf("%s:%d", inst.InternalDomain, inst.InternalPort)
			}
			if inst.PublicDomain != "" {
				network = "Public"
				addr := inst.PublicDomain
				if inst.PublicPort > 0 {
					addr = fmt.Sprintf("%s:%d", inst.PublicDomain, inst.PublicPort)
				}
				if address != "" {
					address = addr + " | " + address
				} else {
					address = addr
				}
			}
			out = append(out, schema.Database{
				InstanceId:    inst.InstanceID,
				Engine:        inst.Engine,
				EngineVersion: inst.EngineVersion,
				Region:        firstNonEmpty(inst.RegionID, region),
				Address:       address,
				NetworkType:   network,
				DBNames:       inst.InstanceName,
			})
		}
		if len(resp.Result.DBInstances) < listPageSize {
			break
		}
	}
	return out, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
