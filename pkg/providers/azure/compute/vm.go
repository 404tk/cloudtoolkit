package compute

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	azapi "github.com/404tk/cloudtoolkit/pkg/providers/azure/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/404tk/cloudtoolkit/utils/logger"
)

type Driver struct {
	Client          *azapi.Client
	SubscriptionIDs []string
}

// GetResource returns all the resources in the store for a provider.
func (d *Driver) GetResource(ctx context.Context) ([]schema.Host, error) {
	list := []schema.Host{}
	logger.Info("List VM ...")

	groupsMap, err := fetchResourceGroups(ctx, d)
	if err != nil {
		logger.Error("Fetch resource groups failed.")
		return list, err
	}

	for subscription, groups := range groupsMap {
		for _, group := range groups {
			vmList, err := fetchVMList(ctx, group, subscription, d.Client)
			if err != nil {
				logger.Error("Fetch VM list failed.")
				return nil, err
			}

			for _, vm := range vmList {
				host := schema.Host{
					ID:       vm.ID,
					State:    vmState(vm),
					HostName: vm.Name,
					Region:   vm.Location,
				}
				if vm.Properties.NetworkProfile == nil {
					list = append(list, host)
					continue
				}

				for _, nic := range vm.Properties.NetworkProfile.NetworkInterfaces {
					nicRes, err := fetchInterfaces(ctx, nic.ID, d.Client)
					if err != nil {
						logger.Error("Fetch interfaces list failed.")
						return list, err
					}
					for _, ipConfig := range nicRes.Properties.IPConfigurations {
						privateIP := strings.TrimSpace(ipConfig.Properties.PrivateIPAddress)
						if privateIP != "" {
							host.PrivateIpv4 = privateIP
						}
						if ipConfig.Properties.PublicIPAddress == nil || strings.TrimSpace(ipConfig.Properties.PublicIPAddress.ID) == "" {
							continue
						}

						publicIP, err := fetchPublicIP(ctx, ipConfig.Properties.PublicIPAddress.ID, d.Client)
						if err != nil {
							continue
						}
						if address := strings.TrimSpace(publicIP.Properties.IPAddress); address != "" {
							host.PublicIPv4 = address
							host.Public = true
							if privateIP != "" {
								host.PrivateIpv4 = privateIP
							}
							break
						}
					}
					if host.Public {
						break
					}
				}

				list = append(list, host)
			}
		}
	}
	return list, nil
}

func fetchResourceGroups(ctx context.Context, sess *Driver) (map[string][]string, error) {
	resGroups := make(map[string][]string, len(sess.SubscriptionIDs))
	for _, subscription := range sess.SubscriptionIDs {
		pager := azapi.NewPager[azapi.ResourceGroup](sess.Client, azapi.Request{
			Method: http.MethodGet,
			Path:   fmt.Sprintf("/subscriptions/%s/resourceGroups", subscription),
			Query:  url.Values{"api-version": {azapi.ResourcesAPIVersion}},
			Idempotent: true,
		})
		items, err := pager.All(ctx)
		if err != nil {
			return resGroups, err
		}
		for _, item := range items {
			if item.Name != "" {
				resGroups[subscription] = append(resGroups[subscription], item.Name)
			}
		}
	}
	return resGroups, nil
}

func fetchVMList(ctx context.Context, group, subscription string, client *azapi.Client) ([]azapi.VirtualMachine, error) {
	pager := azapi.NewPager[azapi.VirtualMachine](client, azapi.Request{
		Method: http.MethodGet,
		Path:   fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines", subscription, group),
		Query:  url.Values{"api-version": {azapi.ComputeAPIVersion}},
		Idempotent: true,
	})
	return pager.All(ctx)
}

func fetchInterfaces(ctx context.Context, nicID string, client *azapi.Client) (azapi.NetworkInterface, error) {
	res, err := azapi.ParseResourceID(nicID)
	if err != nil {
		logger.Error("Parse resource ID failed.")
		return azapi.NetworkInterface{}, err
	}

	var nic azapi.NetworkInterface
	err = client.Do(ctx, azapi.Request{
		Method: http.MethodGet,
		Path: fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkInterfaces/%s",
			res.SubscriptionID,
			res.ResourceGroup,
			res.ResourceName,
		),
		Query:      url.Values{"api-version": {azapi.NetworkAPIVersion}},
		Idempotent: true,
	}, &nic)
	return nic, err
}

func fetchPublicIP(ctx context.Context, publicIPID string, client *azapi.Client) (azapi.PublicIPAddress, error) {
	res, err := azapi.ParseResourceID(publicIPID)
	if err != nil {
		return azapi.PublicIPAddress{}, err
	}

	var ip azapi.PublicIPAddress
	err = client.Do(ctx, azapi.Request{
		Method: http.MethodGet,
		Path: fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/publicIPAddresses/%s",
			res.SubscriptionID,
			res.ResourceGroup,
			res.ResourceName,
		),
		Query:      url.Values{"api-version": {azapi.NetworkAPIVersion}},
		Idempotent: true,
	}, &ip)
	return ip, err
}

func vmState(vm azapi.VirtualMachine) string {
	if state := strings.TrimSpace(vm.Status); state != "" {
		return state
	}
	return strings.TrimSpace(vm.Properties.ProvisioningState)
}
