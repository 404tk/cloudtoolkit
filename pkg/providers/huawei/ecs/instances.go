package ecs

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	_region "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/region"
	ecs "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2/model"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2/region"
)

type Driver struct {
	Auth    basic.Credentials
	Regions []string
}

// GetResource returns all the resources in the store for a provider.
func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := schema.NewResources().Hosts
	logger.Info("Start enumerating ECS ...")
	flag := false
	prevLength := 0
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
			// logger.Error("Enumerate ECS failed.")
			// return list, err
			continue
		}

		for _, instance := range *response.Servers {
			var ipv4, privateIPv4 string
			for _, instance := range instance.Addresses {
				for _, addr := range instance {
					if *addr.OSEXTIPStype == model.GetServerAddressOSEXTIPStypeEnum().FIXED {
						privateIPv4 = addr.Addr
					}
					if *addr.OSEXTIPStype == model.GetServerAddressOSEXTIPStypeEnum().FLOATING {
						ipv4 = addr.Addr
					}
				}
			}
			host := schema.Host{
				HostName:    instance.Name,
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
			prevLength, flag = processbar.RegionPrint(r, len(*response.Servers), prevLength, flag)
		}
	}
done:
	if !flag {
		fmt.Printf("\n\033[F\033[K")
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
