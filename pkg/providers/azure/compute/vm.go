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
	SubscriptionID string
	Authorizer     autorest.Authorizer
}

// GetResource returns all the resources in the store for a provider.
func (d *VmProvider) GetResource(ctx context.Context) ([]*schema.Host, error) {
	list := schema.NewResources().Hosts
	log.Println("Start enumerating VM ...")

	groups, err := fetchResouceGroups(ctx, d)
	if err != nil {
		return list, err
	}

	for _, group := range groups {
		vmList, err := fetchVMList(ctx, group, d)
		if err != nil {
			return nil, err
		}

		for _, vm := range vmList {
			_host := &schema.Host{Region: *vm.Location}
			nics := *vm.NetworkProfile.NetworkInterfaces
			for _, nic := range nics {
				res, err := azure.ParseResourceID(*nic.ID)
				if err != nil {
					return list, err
				}

				nicRes, err := fetchInterfacesList(ctx, group, res.ResourceName, d)
				if err != nil {
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
						return list, err
					}

					publicIP, err := fetchPublicIP(ctx, group, res.ResourceName, d)
					if err != nil {
						return list, err
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
	return list, nil
}

func fetchResouceGroups(ctx context.Context, sess *VmProvider) (resGrpList []string, err error) {
	grClient := resources.NewGroupsClient(sess.SubscriptionID)
	grClient.Authorizer = sess.Authorizer

	for list, err := grClient.ListComplete(ctx, "", nil); list.NotDone(); err = list.Next() {
		if err != nil {
			return nil, err
		}
		resGrp := *list.Value().Name
		resGrpList = append(resGrpList, resGrp)
	}
	return resGrpList, err
}

func fetchVMList(ctx context.Context, group string, sess *VmProvider) (VMList []compute.VirtualMachine, err error) {
	vmClient := compute.NewVirtualMachinesClient(sess.SubscriptionID)
	vmClient.Authorizer = sess.Authorizer

	for vm, err := vmClient.ListComplete(context.Background(), group, ""); vm.NotDone(); err = vm.Next() {
		if err != nil {
			return nil, err
		}
		VMList = append(VMList, vm.Value())
	}

	return VMList, err
}

func fetchInterfacesList(ctx context.Context, group, nic string, sess *VmProvider) (nicRes network.Interface, err error) {
	nicClient := network.NewInterfacesClient(sess.SubscriptionID)
	nicClient.Authorizer = sess.Authorizer
	nicRes, err = nicClient.Get(ctx, group, nic, "")
	return nicRes, err
}

func fetchPublicIP(ctx context.Context, group, publicIP string, sess *VmProvider) (IP network.PublicIPAddress, err error) {
	ipClient := network.NewPublicIPAddressesClient(sess.SubscriptionID)
	ipClient.Authorizer = sess.Authorizer

	IP, err = ipClient.Get(ctx, group, publicIP, "")
	if err != nil {
		return network.PublicIPAddress{}, err
	}

	return IP, err
}
