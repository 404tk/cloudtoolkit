package rds

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	rds "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/rds/v3"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/rds/v3/model"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/rds/v3/region"
)

type Driver struct {
	Auth    basic.Credentials
	Regions []string
}

func (d *Driver) GetDatabases(ctx context.Context) ([]schema.Database, error) {
	list := schema.NewResources().Databases
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("Start enumerating RDS ...")
	}
	client := rds.NewRdsClient(rds.RdsClientBuilder().
		WithRegion(region.ValueOf(d.Regions[0])). // Maybe need traverse region
		WithCredential(d.Auth).
		Build())
	request := &model.ListInstancesRequest{}
	response, err := client.ListInstances(request)
	if err != nil {
		logger.Error("Enumerate RDS failed.")
		return list, err
	}
	for _, instance := range *response.Instances {
		i := reflect.ValueOf(instance.Datastore.Type)
		engine := i.FieldByName("value").String()
		_dbInstance := schema.Database{
			InstanceId:    instance.Id,
			Engine:        engine,
			EngineVersion: instance.Datastore.Version,
			Region:        instance.Region,
		}
		if len(instance.PublicIps) > 0 {
			addrs := []string{}
			for _, ip := range instance.PublicIps {
				addrs = append(addrs, fmt.Sprintf("%s:%d", ip, instance.Port))
			}
			_dbInstance.Address = strings.Join(addrs, "\n")
		} else if len(instance.PrivateIps) > 0 {
			addrs := []string{}
			for _, ip := range instance.PrivateIps {
				addrs = append(addrs, fmt.Sprintf("%s:%d", ip, instance.Port))
			}
			_dbInstance.Address = strings.Join(addrs, "\n")
		}

		list = append(list, _dbInstance)
	}

	return list, nil
}
