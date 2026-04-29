package replay

import "strings"

type vmFixture struct {
	Name           string
	ResourceGroup  string
	Location       string
	State          string
	NICName        string
	PrivateIP      string
	PublicIPName   string
	PublicIP       string
}

var demoVMs = []vmFixture{
	{
		Name:          "ctk-demo-bastion",
		ResourceGroup: demoResourceGroup,
		Location:      demoLocation,
		State:         "Succeeded",
		NICName:       "ctk-demo-bastion-nic",
		PrivateIP:     "10.0.1.10",
		PublicIPName:  "ctk-demo-bastion-pip",
		PublicIP:      "203.0.113.21",
	},
	{
		Name:          "ctk-demo-app",
		ResourceGroup: demoResourceGroup,
		Location:      demoLocation,
		State:         "Succeeded",
		NICName:       "ctk-demo-app-nic",
		PrivateIP:     "10.0.1.11",
	},
}

type storageAccountFixture struct {
	Name           string
	ResourceGroup  string
	Location       string
	BlobServices   []string
	BlobContainers []string
}

var demoStorageAccounts = []storageAccountFixture{
	{
		Name:           "ctkdemologs",
		ResourceGroup:  demoResourceGroup,
		Location:       demoLocation,
		BlobServices:   []string{"default"},
		BlobContainers: []string{"audit", "exports"},
	},
}

func resourceGroupsFor(subscription string) []string {
	if subscription == "" {
		return nil
	}
	groups := map[string]bool{}
	out := make([]string, 0)
	for _, vm := range demoVMs {
		if !groups[vm.ResourceGroup] {
			groups[vm.ResourceGroup] = true
			out = append(out, vm.ResourceGroup)
		}
	}
	for _, sa := range demoStorageAccounts {
		if !groups[sa.ResourceGroup] {
			groups[sa.ResourceGroup] = true
			out = append(out, sa.ResourceGroup)
		}
	}
	return out
}

func vmsForGroup(group string) []vmFixture {
	group = strings.TrimSpace(group)
	out := make([]vmFixture, 0, len(demoVMs))
	for _, vm := range demoVMs {
		if vm.ResourceGroup == group {
			out = append(out, vm)
		}
	}
	return out
}

func storageAccountsForSubscription(subscription string) []storageAccountFixture {
	if subscription == "" {
		return nil
	}
	return append([]storageAccountFixture(nil), demoStorageAccounts...)
}

func storageAccountByName(name string) (storageAccountFixture, bool) {
	name = strings.TrimSpace(name)
	for _, account := range demoStorageAccounts {
		if account.Name == name {
			return account, true
		}
	}
	return storageAccountFixture{}, false
}

func vmByName(name string) (vmFixture, bool) {
	name = strings.TrimSpace(name)
	for _, vm := range demoVMs {
		if vm.Name == name {
			return vm, true
		}
	}
	return vmFixture{}, false
}

func vmByNICName(nicName string) (vmFixture, bool) {
	nicName = strings.TrimSpace(nicName)
	for _, vm := range demoVMs {
		if vm.NICName == nicName {
			return vm, true
		}
	}
	return vmFixture{}, false
}

func vmByPublicIPName(publicIPName string) (vmFixture, bool) {
	publicIPName = strings.TrimSpace(publicIPName)
	for _, vm := range demoVMs {
		if vm.PublicIPName == publicIPName {
			return vm, true
		}
	}
	return vmFixture{}, false
}
