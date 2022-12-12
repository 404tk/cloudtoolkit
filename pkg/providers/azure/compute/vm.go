package compute

import (
	"context"
	"log"

	"github.com/404tk/cloudtoolkit/pkg/schema"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

// VmProvider is an instance provider for Azure API
type VmProvider struct {
	SubscriptionIDs []string
	Authorizer      autorest.Authorizer
}

// GetResource returns all the resources in the store for a provider.
func (d *VmProvider) GetResource(ctx context.Context) ([]*schema.Host, error) {
	list := schema.NewResources().Hosts
	log.Println("[*] Start enumerating VM ...")

	groups_map, err := fetchResouceGroups(ctx, d)
	if err != nil {
		log.Println("[-] Fetch resouce groups failed.")
		return list, err
	}

	for subscription, groups := range groups_map {
		for _, group := range groups {
			vmList, err := fetchVMList(ctx, group, subscription, d.Authorizer)
			if err != nil {
				log.Println("[-] Fetch VM list failed.")
				return nil, err
			}

			for _, vm := range vmList {
				_host := &schema.Host{Region: *vm.Location}
				nics := *vm.NetworkProfile.NetworkInterfaces
				for _, nic := range nics {
					res, err := azure.ParseResourceID(*nic.ID)
					if err != nil {
						log.Println("[-] Parse resource ID failed.")
						return list, err
					}

					nicRes, err := fetchInterfacesList(ctx, group, res.ResourceName, subscription, d.Authorizer)
					if err != nil {
						log.Println("[-] Fetch interfaces list failed.")
						return list, err
					}
					ipConfigs := *nicRes.IPConfigurations
					for _, ipConfig := range ipConfigs {
						ipConfig := *ipConfig.InterfaceIPConfigurationPropertiesFormat
						privateIP := *ipConfig.PrivateIPAddress
						_host.PrivateIpv4 = privateIP

						if ipConfig.PublicIPAddress == nil {
							continue
						}

						res, err := azure.ParseResourceID(*ipConfig.PublicIPAddress.ID)
						if err != nil {
							continue
						}

						publicIP, err := fetchPublicIP(ctx, group, res.ResourceName, subscription, d.Authorizer)
						if err != nil {
							continue
						}

						_host.PublicIPv4 = *publicIP.IPAddress
						_host.PrivateIpv4 = privateIP
						if publicIP.IPAddress != nil {
							_host.Public = true
							break
						}
					}
				}

				list = append(list, _host)
			}
		}
	}
	return list, nil
}

func fetchResouceGroups(ctx context.Context, sess *VmProvider) (map[string][]string, error) {
	resGrp := make(map[string][]string)
	for _, subscription := range sess.SubscriptionIDs {
		grClient := resources.NewGroupsClient(subscription)
		grClient.Authorizer = sess.Authorizer
		resGrp[subscription] = []string{}

		list, err := grClient.List(context.Background(), "", nil)
		if err != nil {
			return resGrp, err
		}
		for _, v := range list.Values() {
			resGrp[subscription] = append(resGrp[subscription], *v.Name)
		}
	}
	return resGrp, nil
}

func fetchVMList(ctx context.Context, group, subscription string, auth autorest.Authorizer) ([]compute.VirtualMachine, error) {
	vmClient := compute.NewVirtualMachinesClient(subscription)
	vmClient.Authorizer = auth
	vm, err := vmClient.List(context.Background(), group, "")
	if err != nil {
		return nil, err
	}

	return vm.Values(), err
}

func fetchInterfacesList(ctx context.Context, group, nic, subscription string, auth autorest.Authorizer) (network.Interface, error) {
	nicClient := network.NewInterfacesClient(subscription)
	nicClient.Authorizer = auth
	nicRes, err := nicClient.Get(ctx, group, nic, "")
	return nicRes, err
}

func fetchPublicIP(ctx context.Context, group, publicIP, subscription string, auth autorest.Authorizer) (IP network.PublicIPAddress, err error) {
	ipClient := network.NewPublicIPAddressesClient(subscription)
	ipClient.Authorizer = auth

	IP, err = ipClient.Get(ctx, group, publicIP, "")
	if err != nil {
		return network.PublicIPAddress{}, err
	}

	return IP, err
}
