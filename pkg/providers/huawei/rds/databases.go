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
	Auth    *basic.Credentials
	Regions []string
}

func (d *Driver) GetDatabases(ctx context.Context) ([]schema.Database, error) {
	list := []schema.Database{}
	select {
	case <-ctx.Done():
		return list, nil
	default:
		logger.Info("List RDS instances ...")
	}
	var regionErrs []string
	for _, r := range d.Regions {
		client, err := newClient(r, d.Auth)
		if err != nil {
			regionErrs = append(regionErrs, err.Error())
			continue
		}
		request := &model.ListInstancesRequest{}
		resp, err := client.ListInstances(request)
		if err != nil {
			regionErrs = append(regionErrs, fmt.Sprintf("%s: %s", r, err))
			continue
		}

		for _, instance := range *resp.Instances {
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
		break
	}

	if len(regionErrs) > 0 {
		return list, fmt.Errorf("%s", strings.Join(regionErrs, "; "))
	}
	return list, nil
}

func newClient(r string, auth *basic.Credentials) (client *rds.RdsClient, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("unsupported RDS region %q: %v", r, rec)
			client = nil
		}
	}()

	client = rds.NewRdsClient(rds.RdsClientBuilder().
		WithRegion(region.ValueOf(r)).
		WithCredential(auth).
		Build())
	return client, nil
}
