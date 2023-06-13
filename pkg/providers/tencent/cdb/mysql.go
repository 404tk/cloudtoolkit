package cdb

import (
	"context"
	"fmt"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/processbar"
	cdb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cdb/v20170320"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

type CdbProvider struct {
	Credential *common.Credential
	Region     string
}

func (d *CdbProvider) ListMySQL(ctx context.Context) ([]*schema.Database, error) {
	list := schema.NewResources().Databases
	select {
	case <-ctx.Done():
		return list, nil
	default:
		log.Println("[*] Start enumerating MySQL ...")
	}
	cpf := profile.NewClientProfile()
	var regions []string
	if d.Region == "all" {
		client, _ := cdb.NewClient(d.Credential, "ap-guangzhou", cpf)
		req := cdb.NewDescribeCdbZoneConfigRequest()
		resp, err := client.DescribeCdbZoneConfig(req)
		if err != nil {
			log.Println("[-] Enumerate MySQL failed.")
			return list, err
		}
		for _, r := range resp.Response.DataResult.Regions {
			regions = append(regions, *r.Region)
		}
	} else {
		regions = append(regions, d.Region)
	}

	flag := false
	prevLength := 0
	for _, r := range regions {
		client, _ := cdb.NewClient(d.Credential, r, cpf)
		request := cdb.NewDescribeDBInstancesRequest()
		response, err := client.DescribeDBInstances(request)
		if err != nil {
			log.Println("[-] Enumerate MySQL failed.")
			return list, err
		}

		for _, instance := range response.Response.Items {
			_db := &schema.Database{
				DBInstanceId:  *instance.InstanceId,
				Engine:        "MySQL",
				EngineVersion: *instance.EngineVersion,
				Region:        *instance.Region,
			}
			list = append(list, _db)
		}
		select {
		case <-ctx.Done():
			goto done
		default:
			prevLength, flag = processbar.RegionPrint(r, len(response.Response.Items), prevLength, flag)
		}
	}
done:
	if !flag {
		fmt.Printf("\n\033[F\033[K")
	}
	return list, nil
}
