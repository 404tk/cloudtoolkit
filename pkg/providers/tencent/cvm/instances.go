package cvm

import (
	"context"
	"fmt"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/processbar"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

type Driver struct {
	Credential *common.Credential
	Region     string
}

// GetResource returns all the resources in the store for a provider.
func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := schema.NewResources().Hosts
	log.Println("[*] Start enumerating CVM ...")
	cpf := profile.NewClientProfile()
	var regions []string
	if d.Region == "all" {
		client, _ := cvm.NewClient(d.Credential, "ap-guangzhou", cpf)
		req := cvm.NewDescribeRegionsRequest()
		resp, err := client.DescribeRegions(req)
		if err != nil {
			log.Println("[-] Enumerate CVM failed.")
			return list, err
		}
		for _, r := range resp.Response.RegionSet {
			regions = append(regions, *r.Region)
		}
	} else {
		regions = append(regions, d.Region)
	}
	flag := false
	prevLength := 0
	for _, r := range regions {
		client, _ := cvm.NewClient(d.Credential, r, cpf)
		request := cvm.NewDescribeInstancesRequest()
		response, err := client.DescribeInstances(request)
		if err != nil {
			log.Println("[-] Enumerate CVM failed.")
			return list, err
		}

		for _, instance := range response.Response.InstanceSet {
			var ipv4, privateIPv4 string
			if len(instance.PublicIpAddresses) > 0 {
				ipv4 = *instance.PublicIpAddresses[0]
			}
			if len(instance.PrivateIpAddresses) > 0 {
				privateIPv4 = *instance.PrivateIpAddresses[0]
			}
			host := schema.Host{
				HostName:    *instance.InstanceName,
				PublicIPv4:  ipv4,
				PrivateIpv4: privateIPv4,
				Public:      ipv4 != "",
				Region:      r,
			}
			list = append(list, host)
		}
		select {
		case <-ctx.Done():
			goto done
		default:
			prevLength, flag = processbar.RegionPrint(r, len(response.Response.InstanceSet), prevLength, flag)
		}
	}
done:
	if !flag {
		fmt.Printf("\n\033[F\033[K")
	}

	return list, nil
}
