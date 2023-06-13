package cdb

import (
	"context"
	"fmt"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/processbar"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	mariadb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/mariadb/v20170312"
)

func (d *CdbProvider) ListMariaDB(ctx context.Context) ([]*schema.Database, error) {
	list := schema.NewResources().Databases
	select {
	case <-ctx.Done():
		return list, nil
	default:
		log.Println("[*] Start enumerating MariaDB ...")
	}
	cpf := profile.NewClientProfile()
	var regions []string
	if d.Region == "all" {
		client, _ := mariadb.NewClient(d.Credential, "ap-guangzhou", cpf)
		req := mariadb.NewDescribeSaleInfoRequest()
		resp, err := client.DescribeSaleInfo(req)
		if err != nil {
			log.Println("[-] Enumerate MariaDB failed.")
			return list, err
		}
		for _, r := range resp.Response.RegionList {
			regions = append(regions, *r.Region)
		}
	} else {
		regions = append(regions, d.Region)
	}

	flag := false
	prevLength := 0
	for _, r := range regions {
		client, _ := mariadb.NewClient(d.Credential, r, cpf)
		request := mariadb.NewDescribeDBInstancesRequest()
		response, err := client.DescribeDBInstances(request)
		if err != nil {
			log.Println("[-] Enumerate MariaDB failed.")
			return list, err
		}

		for _, instance := range response.Response.Instances {
			_db := &schema.Database{
				DBInstanceId:  *instance.InstanceId,
				Engine:        "MariaDB",
				EngineVersion: *instance.DbVersion,
				Region:        *instance.Region,
			}
			list = append(list, _db)
		}
		select {
		case <-ctx.Done():
			goto done
		default:
			prevLength, flag = processbar.RegionPrint(r, len(response.Response.Instances), prevLength, flag)
		}
	}
done:
	if !flag {
		fmt.Printf("\n\033[F\033[K")
	}
	return list, nil
}
