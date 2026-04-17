package ecs

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/endpoint"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/runtime/regionrun"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
	"github.com/404tk/cloudtoolkit/utils/processbar"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	ecs "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2/model"
)

type Driver struct {
	Auth    *basic.Credentials
	Regions []string
}

// GetResource returns all the resources in the store for a provider.
func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := []schema.Host{}
	logger.Info("List ECS instances ...")
	tracker := processbar.NewRegionTracker()
	defer tracker.Finish()
	var regionErrs []string
	var errMu sync.Mutex
	got, _ := regionrun.ForEach(ctx, d.Regions, 0, tracker, func(ctx context.Context, r string) ([]schema.Host, error) {
		client := newClient(r, d.Auth)
		const limit = int32(100)
		items, err := paginate.Fetch(ctx, func(ctx context.Context, page int32) (paginate.Page[schema.Host, int32], error) {
			if page == 0 {
				page = 1
			}
			pagePtr := page
			limitPtr := limit
			response, err := client.ListServersDetails(&model.ListServersDetailsRequest{Limit: &limitPtr, Offset: &pagePtr})
			if err != nil {
				return paginate.Page[schema.Host, int32]{}, err
			}
			var items []schema.Host
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
				items = append(items, schema.Host{
					State:       instance.Status,
					HostName:    instance.Name,
					PublicIPv4:  ipv4,
					PrivateIpv4: privateIPv4,
					Public:      ipv4 != "",
					Region:      r,
				})
			}
			return paginate.Page[schema.Host, int32]{
				Items: items,
				Next:  page + 1,
				Done:  page*limit >= *response.Count,
			}, nil
		})
		if err != nil {
			errMu.Lock()
			regionErrs = append(regionErrs, fmt.Sprintf("%s: %s", r, err))
			errMu.Unlock()
		}
		return items, nil
	})
	list = append(list, got...)

	if len(regionErrs) > 0 {
		return list, fmt.Errorf("%s", strings.Join(regionErrs, "; "))
	}
	return list, nil
}

func newClient(r string, auth *basic.Credentials) *ecs.EcsClient {
	return ecs.NewEcsClient(ecs.EcsClientBuilder().
		WithEndpoint(endpoint.For("ecs", r, false)).
		WithCredential(auth).
		Build())
}
