package rds

import (
	"context"
	"log"
	"reflect"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	rds "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/rds/v3"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/rds/v3/model"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/rds/v3/region"
)

type RdsProvider struct {
	Auth    basic.Credentials
	Regions []string
}

func (d *RdsProvider) GetDatabases(ctx context.Context) ([]*schema.Database, error) {
	list := schema.NewResources().Databases
	select {
	case <-ctx.Done():
		return list, nil
	default:
		log.Println("[*] Start enumerating RDS ...")
	}
	client := rds.NewRdsClient(rds.RdsClientBuilder().
		WithRegion(region.ValueOf(d.Regions[0])). // Maybe need traverse region
		WithCredential(d.Auth).
		Build())
	request := &model.ListInstancesRequest{}
	response, err := client.ListInstances(request)
	if err != nil {
		log.Println("[-] Enumerate RDS failed.")
		return list, err
	}
	for _, instance := range *response.Instances {
		i := reflect.ValueOf(instance.Datastore.Type)
		engine := i.FieldByName("value").String()
		_dbInstance := &schema.Database{
			DBInstanceId:  instance.Id,
			Engine:        engine,
			EngineVersion: instance.Datastore.Version,
			Region:        instance.Region,
		}

		list = append(list, _dbInstance)
	}

	return list, nil
}
