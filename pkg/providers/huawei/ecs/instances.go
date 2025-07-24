package ecs

import (
	"context"
	"fmt"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
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
	list := []schema.Host{}
	logger.Info("List ECS instances ...")
	flag := false
	prevLength := 0
	for _, r := range d.Regions {
		client := newClient(r, d.Auth)
		if client == nil {
			continue
		}

		count := 0
		limitRequest := int32(100)
		page := int32(1)
		request := &model.ListServersDetailsRequest{Limit: &limitRequest}
		for {
			request.Offset = &page
			response, err := client.ListServersDetails(request)
			if err != nil {
				break
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
					State:       instance.Status,
					HostName:    instance.Name,
					PublicIPv4:  ipv4,
					PrivateIpv4: privateIPv4,
					Public:      ipv4 != "",
					Region:      r,
				}
				list = append(list, host)
			}
			if page*limitRequest >= *response.Count {
				count = int(*response.Count)
				break
			}
			page++
		}
		select {
		case <-ctx.Done():
			goto done
		default:
			prevLength, flag = processbar.RegionPrint(r, count, prevLength, flag)
		}
	}
done:
	if !flag {
		fmt.Printf("\n\033[F\033[K")
	}

	return list, nil
}

func newClient(r string, auth basic.Credentials) *ecs.EcsClient {
	defer func() {
		if err := recover(); err != nil {
			return
		}
	}()
	client := ecs.NewEcsClient(ecs.EcsClientBuilder().
		WithRegion(region.ValueOf(r)).
		WithCredential(auth).
		Build())
	return client
}
