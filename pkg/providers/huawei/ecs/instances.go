package ecs

import (
	"context"
	"fmt"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	_region "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/region"
	ecs "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2/model"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2/region"
)

type InstanceProvider struct {
	Auth    basic.Credentials
	Regions []string
}

// GetResource returns all the resources in the store for a provider.
func (d *InstanceProvider) GetResource(ctx context.Context) ([]*schema.Host, error) {
	list := schema.NewResources().Hosts
	log.Println("[*] Start enumerating ECS ...")
	for _, r := range d.Regions {
		_r := getRegion(r)
		if _r == nil {
			continue
		}
		client := newClient(_r, d.Auth)
		if client == nil {
			continue
		}

		request := &model.ListServersDetailsRequest{}
		response, err := client.ListServersDetails(request)
		if err != nil {
			log.Println("[-] Enumerate ECS failed.")
			return list, err
		}

		for _, instances := range *response.Servers {
			var ipv4, privateIPv4 string
			for _, instance := range instances.Addresses {
				for _, addr := range instance {
					if *addr.OSEXTIPStype == model.GetServerAddressOSEXTIPStypeEnum().FIXED {
						privateIPv4 = addr.Addr
					}
					if *addr.OSEXTIPStype == model.GetServerAddressOSEXTIPStypeEnum().FLOATING {
						ipv4 = addr.Addr
					}
				}
			}
			host := &schema.Host{
				PublicIPv4:  ipv4,
				PrivateIpv4: privateIPv4,
				Public:      ipv4 != "",
				Region:      r,
			}
			list = append(list, host)
		}
		progress := fmt.Sprintf("Inquiring %s regionId,number of discovered hosts: %d", r, len(*response.Servers))
		log.Println(progress)
	}

	return list, nil
}

func newClient(r *_region.Region, auth basic.Credentials) *ecs.EcsClient {
	defer func() {
		if err := recover(); err != nil {
			// log.Printf("%s: %v\n", r.Id, err)
			return
		}
	}()
	client := ecs.NewEcsClient(ecs.EcsClientBuilder().
		WithRegion(r).
		WithCredential(auth).
		Build())
	return client
}

func getRegion(r string) *_region.Region {
	defer func() {
		if err := recover(); err != nil {
			return
		}
	}()
	return region.ValueOf(r)
}
