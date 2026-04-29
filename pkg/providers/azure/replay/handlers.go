package replay

import (
	"fmt"
	"net/http"
	"strings"

	azapi "github.com/404tk/cloudtoolkit/pkg/providers/azure/api"
)

func (t *transport) handleARM(req *http.Request) (*http.Response, error) {
	path := strings.Trim(req.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return armErrorResponse(req, http.StatusNotFound, "InvalidPath",
			"unsupported arm path"), nil
	}

	if path == "subscriptions" {
		return t.handleListSubscriptions(req)
	}
	if len(parts) >= 1 && parts[0] == "subscriptions" {
		if len(parts) < 2 {
			return armErrorResponse(req, http.StatusBadRequest, "InvalidParameter",
				"missing subscription id"), nil
		}
		subscription := parts[1]
		if subscription != demoSubscriptionID {
			return armErrorResponse(req, http.StatusForbidden, "SubscriptionNotFound",
				fmt.Sprintf("subscription %s not visible to current credentials", subscription)), nil
		}
		rest := parts[2:]
		return t.routeSubscriptionScoped(req, subscription, rest)
	}
	return armErrorResponse(req, http.StatusNotFound, "InvalidPath",
		fmt.Sprintf("unsupported arm path: %s", path)), nil
}

func (t *transport) routeSubscriptionScoped(req *http.Request, subscription string, parts []string) (*http.Response, error) {
	if len(parts) == 0 {
		return armErrorResponse(req, http.StatusNotFound, "InvalidPath",
			"unsupported subscription-scoped path"), nil
	}
	switch parts[0] {
	case "resourceGroups":
		if len(parts) == 1 {
			return t.handleListResourceGroups(req, subscription)
		}
		group := parts[1]
		if len(parts) >= 4 && parts[2] == "providers" {
			return t.handleResourceGroupProvider(req, subscription, group, parts[3:])
		}
		return armErrorResponse(req, http.StatusNotFound, "InvalidPath",
			fmt.Sprintf("unsupported resource group path: %v", parts)), nil
	case "providers":
		return t.handleSubscriptionProvider(req, subscription, parts[1:])
	}
	return armErrorResponse(req, http.StatusNotFound, "InvalidPath",
		fmt.Sprintf("unsupported subscription path: %v", parts)), nil
}

func (t *transport) handleListSubscriptions(req *http.Request) (*http.Response, error) {
	resp := azapi.ListSubscriptionsResponse{
		Value: []azapi.Subscription{{
			SubscriptionID: demoSubscriptionID,
			DisplayName:    demoSubscriptionDN,
			State:          "Enabled",
		}},
	}
	return jsonResponse(req, resp), nil
}

func (t *transport) handleListResourceGroups(req *http.Request, subscription string) (*http.Response, error) {
	resp := azapi.ListResourceGroupsResponse{}
	for _, group := range resourceGroupsFor(subscription) {
		resp.Value = append(resp.Value, azapi.ResourceGroup{
			ID:       fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", subscription, group),
			Name:     group,
			Location: demoLocation,
		})
	}
	return jsonResponse(req, resp), nil
}

func (t *transport) handleResourceGroupProvider(req *http.Request, subscription, group string, parts []string) (*http.Response, error) {
	if len(parts) < 2 {
		return armErrorResponse(req, http.StatusNotFound, "InvalidPath",
			fmt.Sprintf("unsupported provider path: %v", parts)), nil
	}
	provider := parts[0]
	rest := parts[1:]
	switch {
	case strings.EqualFold(provider, "Microsoft.Compute") && len(rest) >= 1 && rest[0] == "virtualMachines":
		if len(rest) == 1 {
			return t.handleListVMs(req, subscription, group)
		}
		return armErrorResponse(req, http.StatusNotFound, "InvalidPath",
			fmt.Sprintf("unsupported VM subpath: %v", rest)), nil
	case strings.EqualFold(provider, "Microsoft.Network") && len(rest) >= 2 && rest[0] == "networkInterfaces":
		return t.handleShowNIC(req, subscription, group, rest[1])
	case strings.EqualFold(provider, "Microsoft.Network") && len(rest) >= 2 && rest[0] == "publicIPAddresses":
		return t.handleShowPublicIP(req, subscription, group, rest[1])
	case strings.EqualFold(provider, "Microsoft.Storage") && len(rest) >= 2 && rest[0] == "storageAccounts":
		return t.handleStorageScoped(req, subscription, group, rest[1:])
	}
	return armErrorResponse(req, http.StatusNotFound, "InvalidPath",
		fmt.Sprintf("unsupported provider path: %s/%v", provider, rest)), nil
}

func (t *transport) handleSubscriptionProvider(req *http.Request, subscription string, parts []string) (*http.Response, error) {
	if len(parts) < 2 {
		return armErrorResponse(req, http.StatusNotFound, "InvalidPath",
			fmt.Sprintf("unsupported subscription-provider path: %v", parts)), nil
	}
	provider := parts[0]
	rest := parts[1:]
	switch {
	case strings.EqualFold(provider, "Microsoft.Storage") && len(rest) >= 1 && rest[0] == "storageAccounts":
		return t.handleListStorageAccounts(req, subscription)
	}
	return armErrorResponse(req, http.StatusNotFound, "InvalidPath",
		fmt.Sprintf("unsupported subscription provider: %s/%v", provider, rest)), nil
}

func (t *transport) handleListVMs(req *http.Request, subscription, group string) (*http.Response, error) {
	resp := azapi.ListVirtualMachinesResponse{}
	for _, vm := range vmsForGroup(group) {
		nicRef := azapi.VMNetworkInterfaceRef{
			ID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkInterfaces/%s",
				subscription, group, vm.NICName),
		}
		resp.Value = append(resp.Value, azapi.VirtualMachine{
			ID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s",
				subscription, group, vm.Name),
			Name:     vm.Name,
			Location: vm.Location,
			Properties: azapi.VirtualMachineProps{
				ProvisioningState: vm.State,
				NetworkProfile: &azapi.VMNetworkProfile{
					NetworkInterfaces: []azapi.VMNetworkInterfaceRef{nicRef},
				},
			},
		})
	}
	return jsonResponse(req, resp), nil
}

func (t *transport) handleShowNIC(req *http.Request, subscription, group, nicName string) (*http.Response, error) {
	vm, ok := vmByNICName(nicName)
	if !ok {
		return armErrorResponse(req, http.StatusNotFound, "ResourceNotFound",
			fmt.Sprintf("network interface %s not found", nicName)), nil
	}
	ipConfig := azapi.IPConfiguration{
		Name: "ipconfig1",
		Properties: azapi.IPConfigurationProps{
			PrivateIPAddress: vm.PrivateIP,
		},
	}
	if vm.PublicIPName != "" {
		ipConfig.Properties.PublicIPAddress = &azapi.ResourceRef{
			ID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/publicIPAddresses/%s",
				subscription, group, vm.PublicIPName),
		}
	}
	resp := azapi.NetworkInterface{
		ID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkInterfaces/%s",
			subscription, group, nicName),
		Name: nicName,
		Properties: azapi.NetworkInterfaceProps{
			IPConfigurations: []azapi.IPConfiguration{ipConfig},
		},
	}
	return jsonResponse(req, resp), nil
}

func (t *transport) handleShowPublicIP(req *http.Request, subscription, group, name string) (*http.Response, error) {
	vm, ok := vmByPublicIPName(name)
	if !ok {
		return armErrorResponse(req, http.StatusNotFound, "ResourceNotFound",
			fmt.Sprintf("public IP %s not found", name)), nil
	}
	resp := azapi.PublicIPAddress{
		ID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/publicIPAddresses/%s",
			subscription, group, name),
		Name: name,
		Properties: azapi.PublicIPAddressProps{
			IPAddress: vm.PublicIP,
		},
	}
	return jsonResponse(req, resp), nil
}

func (t *transport) handleListStorageAccounts(req *http.Request, subscription string) (*http.Response, error) {
	resp := azapi.ListStorageAccountsResponse{}
	for _, account := range storageAccountsForSubscription(subscription) {
		resp.Value = append(resp.Value, azapi.StorageAccount{
			ID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s",
				subscription, account.ResourceGroup, account.Name),
			Name:     account.Name,
			Location: account.Location,
		})
	}
	return jsonResponse(req, resp), nil
}

func (t *transport) handleStorageScoped(req *http.Request, subscription, group string, parts []string) (*http.Response, error) {
	if len(parts) < 1 {
		return armErrorResponse(req, http.StatusNotFound, "InvalidPath",
			fmt.Sprintf("unsupported storage path: %v", parts)), nil
	}
	accountName := parts[0]
	account, ok := storageAccountByName(accountName)
	if !ok || account.ResourceGroup != group {
		return armErrorResponse(req, http.StatusNotFound, "StorageAccountNotFound",
			fmt.Sprintf("storage account %s not found in %s", accountName, group)), nil
	}
	rest := parts[1:]
	if len(rest) >= 1 && rest[0] == "blobServices" {
		// blobServices, blobServices/default, blobServices/default/containers
		switch {
		case len(rest) == 1:
			return t.handleListBlobServices(req, subscription, group, account)
		case len(rest) >= 3 && rest[2] == "containers":
			return t.handleListBlobContainers(req, subscription, group, account, rest[1])
		}
	}
	return armErrorResponse(req, http.StatusNotFound, "InvalidPath",
		fmt.Sprintf("unsupported storage subpath: %v", rest)), nil
}

func (t *transport) handleListBlobServices(req *http.Request, subscription, group string, account storageAccountFixture) (*http.Response, error) {
	resp := azapi.ListBlobServicesResponse{}
	for _, name := range account.BlobServices {
		resp.Value = append(resp.Value, azapi.BlobService{
			ID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s/blobServices/%s",
				subscription, group, account.Name, name),
			Name: name,
		})
	}
	return jsonResponse(req, resp), nil
}

func (t *transport) handleListBlobContainers(req *http.Request, subscription, group string, account storageAccountFixture, serviceName string) (*http.Response, error) {
	resp := azapi.ListBlobContainersResponse{}
	for _, name := range account.BlobContainers {
		resp.Value = append(resp.Value, azapi.BlobContainer{
			ID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s/blobServices/%s/containers/%s",
				subscription, group, account.Name, serviceName, name),
			Name: name,
		})
	}
	return jsonResponse(req, resp), nil
}
