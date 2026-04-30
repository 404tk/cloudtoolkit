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

// demoRoleDefinitions is a small, fixed catalog of the most commonly assigned
// built-in roles. The replay returns only entries with names that match the
// `$filter=roleName eq '...'` filter on roleDefinitions.
type roleDefinitionFixture struct {
	Name string
	GUID string
}

var demoRoleDefinitions = []roleDefinitionFixture{
	{Name: "Reader", GUID: "acdd72a7-3385-48ef-bd42-f606fba81ae7"},
	{Name: "Contributor", GUID: "b24988ac-6180-42a0-ab88-20f7382dd24c"},
	{Name: "Owner", GUID: "8e3af657-a8ff-443c-a75c-2fe8c4bcb635"},
	{Name: "Storage Blob Data Reader", GUID: "2a2b9908-6ea1-4ae2-8e65-a410df84e7d1"},
	{Name: "Storage Blob Data Contributor", GUID: "ba92f5b4-2d11-453d-a403-e96b0029c9fe"},
}

func roleDefinitionByName(name string) (roleDefinitionFixture, bool) {
	name = strings.TrimSpace(name)
	for _, def := range demoRoleDefinitions {
		if strings.EqualFold(def.Name, name) {
			return def, true
		}
	}
	return roleDefinitionFixture{}, false
}

func roleDefinitionByGUID(guid string) (roleDefinitionFixture, bool) {
	guid = strings.TrimSpace(guid)
	for _, def := range demoRoleDefinitions {
		if strings.EqualFold(def.GUID, guid) {
			return def, true
		}
	}
	return roleDefinitionFixture{}, false
}

// demoPrincipals are the only object IDs the replay accepts as PUT body
// principalId. Any other value comes back as PrincipalNotFound, mirroring real
// Azure behavior.
var demoPrincipals = []string{
	"11111111-2222-3333-4444-555555555555",
	"22222222-3333-4444-5555-666666666666",
	"33333333-4444-5555-6666-777777777777",
}

func isKnownPrincipal(id string) bool {
	id = strings.TrimSpace(id)
	for _, p := range demoPrincipals {
		if strings.EqualFold(p, id) {
			return true
		}
	}
	return false
}

// roleAssignmentFixture is the shape stored both for the seed list and for
// transport-state assignments created during a replay session.
type roleAssignmentFixture struct {
	Name             string
	PrincipalID      string
	RoleDefinitionID string
	Scope            string
}

// demoRoleAssignments is the seeded list returned at the start of a replay
// session. Newly-created assignments are stored on the transport state.
// Scope of "" means "default subscription scope".
var demoRoleAssignments = []roleAssignmentFixture{
	{
		Name:             "11112222-3333-4444-5555-666677778888",
		PrincipalID:      "11111111-2222-3333-4444-555555555555",
		RoleDefinitionID: "acdd72a7-3385-48ef-bd42-f606fba81ae7",
		Scope:            "",
	},
	{
		Name:             "aaaa1111-bbbb-2222-cccc-3333dddd4444",
		PrincipalID:      "22222222-3333-4444-5555-666666666666",
		RoleDefinitionID: "2a2b9908-6ea1-4ae2-8e65-a410df84e7d1",
		Scope:            "",
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

// containerACLKey is the join of resource group, account, and container name
// used as the override map key. It must match the path-derived form used by
// the PATCH/GET handlers.
func containerACLKey(group, account, container string) string {
	return group + "/" + account + "/" + container
}
