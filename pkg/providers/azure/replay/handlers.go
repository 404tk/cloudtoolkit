package replay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	azapi "github.com/404tk/cloudtoolkit/pkg/providers/azure/api"
	demoreplay "github.com/404tk/cloudtoolkit/pkg/providers/replay"
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
		if len(rest) >= 3 && rest[2] == "runCommand" {
			return t.handleVMRunCommand(req, subscription, group, rest[1])
		}
		return armErrorResponse(req, http.StatusNotFound, "InvalidPath",
			fmt.Sprintf("unsupported VM subpath: %v", rest)), nil
	case strings.EqualFold(provider, "Microsoft.Network") && len(rest) >= 2 && rest[0] == "networkInterfaces":
		return t.handleShowNIC(req, subscription, group, rest[1])
	case strings.EqualFold(provider, "Microsoft.Network") && len(rest) >= 2 && rest[0] == "publicIPAddresses":
		return t.handleShowPublicIP(req, subscription, group, rest[1])
	case strings.EqualFold(provider, "Microsoft.Storage") && len(rest) >= 2 && rest[0] == "storageAccounts":
		return t.handleStorageScoped(req, subscription, group, rest[1:])
	case strings.EqualFold(provider, "Microsoft.Sql") && len(rest) >= 2 && rest[0] == "servers":
		return t.handleSQLServer(req, subscription, group, rest[1])
	}
	return armErrorResponse(req, http.StatusNotFound, "InvalidPath",
		fmt.Sprintf("unsupported provider path: %s/%v", provider, rest)), nil
}

func (t *transport) handleSQLServer(req *http.Request, subscription, group, server string) (*http.Response, error) {
	switch req.Method {
	case http.MethodPatch:
		// PATCH succeeds with the server resource shape; ARM normally returns
		// 200 + updated body or 202 + Location for async.
		resp := azapi.SQLServer{
			ID:       fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Sql/servers/%s", subscription, group, server),
			Name:     server,
			Location: demoLocation,
			Properties: azapi.SQLServerProperties{
				AdministratorLogin:       "ctkadmin",
				FullyQualifiedDomainName: server + ".database.windows.net",
				State:                    "Ready",
				Version:                  "12.0",
			},
		}
		return jsonResponse(req, resp), nil
	}
	return armErrorResponse(req, http.StatusMethodNotAllowed, "MethodNotAllowed",
		"unsupported sql server method"), nil
}

func (t *transport) handleVMRunCommand(req *http.Request, subscription, group, vmName string) (*http.Response, error) {
	if req.Method != http.MethodPost {
		return armErrorResponse(req, http.StatusMethodNotAllowed, "MethodNotAllowed",
			"unsupported runCommand method"), nil
	}
	resp := azapi.RunCommandResult{Value: []azapi.RunCommandInstanceView{{
		Code:          "ProvisioningState/succeeded",
		Level:         "Info",
		DisplayStatus: "Provisioning succeeded",
		Message:       fmt.Sprintf("[stdout]\nctk-demo-vm-output: subscription=%s rg=%s vm=%s\n[stderr]\n", subscription, group, vmName),
	}}}
	return jsonResponse(req, resp), nil
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
	case strings.EqualFold(provider, "Microsoft.Authorization") && len(rest) >= 1 && rest[0] == "roleAssignments":
		return t.handleRoleAssignments(req, subscription, rest[1:])
	case strings.EqualFold(provider, "Microsoft.Authorization") && len(rest) >= 1 && rest[0] == "roleDefinitions":
		return t.handleRoleDefinitions(req, subscription, rest[1:])
	case strings.EqualFold(provider, "Microsoft.Insights") && len(rest) >= 3 && rest[0] == "eventtypes" && rest[1] == "management" && rest[2] == "values":
		return t.handleActivityLog(req, subscription)
	}
	return armErrorResponse(req, http.StatusNotFound, "InvalidPath",
		fmt.Sprintf("unsupported subscription provider: %s/%v", provider, rest)), nil
}

func (t *transport) handleActivityLog(req *http.Request, subscription string) (*http.Response, error) {
	resp := azapi.ActivityLogResponse{}
	resp.Value = demoAzureActivityEvents(subscription)
	return jsonResponse(req, resp), nil
}

func demoAzureActivityEvents(subscription string) []azapi.ActivityLogEvent {
	groups := resourceGroupsFor(subscription)
	group := "ctk-demo-rg"
	if len(groups) > 0 {
		group = groups[0]
	}
	resourceID := func(provider, name string) string {
		return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/%s", subscription, group, provider, name)
	}
	mk := func(id, op, opLocal, ts, caller, ip, status, scope, rt string) azapi.ActivityLogEvent {
		ev := azapi.ActivityLogEvent{
			EventDataID:         id,
			EventTimestamp:      ts,
			Caller:              caller,
			ResourceID:          scope,
			SubmissionTimestamp: ts,
		}
		ev.OperationName.Value = op
		ev.OperationName.LocalizedValue = opLocal
		ev.HTTPRequest.ClientIPAddress = ip
		ev.Status.Value = status
		ev.Status.LocalizedValue = status
		ev.Authorization.Action = op
		ev.Authorization.Scope = scope
		ev.ResourceType.Value = rt
		ev.ResourceType.LocalizedValue = rt
		return ev
	}
	return []azapi.ActivityLogEvent{
		mk(
			"evt-azure-0001",
			"Microsoft.Authorization/roleAssignments/write",
			"Create role assignment",
			"2026-04-22T09:11:00.0000000Z",
			"ctk-demo-app",
			"203.0.113.81",
			"Succeeded",
			resourceID("Microsoft.Authorization", "roleAssignments/00000000-0000-0000-0000-000000000001"),
			"Microsoft.Authorization/roleAssignments",
		),
		mk(
			"evt-azure-0002",
			"Microsoft.Storage/storageAccounts/blobServices/containers/write",
			"Update blob container",
			"2026-04-22T09:14:30.0000000Z",
			"ctk-demo-app",
			"203.0.113.81",
			"Succeeded",
			resourceID("Microsoft.Storage", "storageAccounts/ctkdemoblobs/blobServices/default/containers/public-export"),
			"Microsoft.Storage/storageAccounts/blobServices/containers",
		),
		mk(
			"evt-azure-0003",
			"Microsoft.Compute/virtualMachines/runCommand/action",
			"Run command on VM",
			"2026-04-22T09:18:42.0000000Z",
			"ctk-demo-app",
			"203.0.113.81",
			"Failed",
			resourceID("Microsoft.Compute", "virtualMachines/ctk-demo-vm-01"),
			"Microsoft.Compute/virtualMachines",
		),
	}
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
		// blobServices, blobServices/default, blobServices/default/containers, blobServices/default/containers/{name}
		switch {
		case len(rest) == 1:
			return t.handleListBlobServices(req, subscription, group, account)
		case len(rest) == 3 && rest[2] == "containers":
			return t.handleListBlobContainers(req, subscription, group, account, rest[1])
		case len(rest) == 4 && rest[2] == "containers":
			return t.handleContainerACL(req, subscription, group, account, rest[1], rest[3])
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
		level := t.lookupContainerACL(group, account.Name, name)
		resp.Value = append(resp.Value, azapi.BlobContainer{
			ID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s/blobServices/%s/containers/%s",
				subscription, group, account.Name, serviceName, name),
			Name: name,
			Properties: &azapi.BlobContainerProperties{
				PublicAccess: level,
			},
		})
	}
	return jsonResponse(req, resp), nil
}

func (t *transport) handleContainerACL(req *http.Request, subscription, group string, account storageAccountFixture, serviceName, container string) (*http.Response, error) {
	if !containerExists(account, container) {
		return armErrorResponse(req, http.StatusNotFound, "ContainerNotFound",
			fmt.Sprintf("container %s not found in account %s", container, account.Name)), nil
	}
	switch req.Method {
	case http.MethodGet:
		level := t.lookupContainerACL(group, account.Name, container)
		resp := azapi.BlobContainer{
			ID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s/blobServices/%s/containers/%s",
				subscription, group, account.Name, serviceName, container),
			Name: container,
			Properties: &azapi.BlobContainerProperties{
				PublicAccess: level,
			},
		}
		return jsonResponse(req, resp), nil
	case http.MethodPatch, http.MethodPut:
		body, err := readBody(req)
		if err != nil {
			return armErrorResponse(req, http.StatusBadRequest, "InvalidRequestBody", err.Error()), nil
		}
		var patch azapi.BlobContainerPatchRequest
		if err := json.Unmarshal(body, &patch); err != nil {
			return armErrorResponse(req, http.StatusBadRequest, "InvalidRequestBody", err.Error()), nil
		}
		level := strings.TrimSpace(patch.Properties.PublicAccess)
		if level == "" {
			level = "None"
		}
		switch level {
		case "None", "Blob", "Container":
		default:
			return armErrorResponse(req, http.StatusBadRequest, "InvalidParameter",
				fmt.Sprintf("publicAccess %q is not one of None / Blob / Container", level)), nil
		}
		t.setContainerACL(group, account.Name, container, level)
		resp := azapi.BlobContainer{
			ID: fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s/blobServices/%s/containers/%s",
				subscription, group, account.Name, serviceName, container),
			Name: container,
			Properties: &azapi.BlobContainerProperties{
				PublicAccess: level,
			},
		}
		return jsonResponse(req, resp), nil
	}
	return armErrorResponse(req, http.StatusMethodNotAllowed, "MethodNotAllowed",
		fmt.Sprintf("method %s not supported on container", req.Method)), nil
}

func (t *transport) lookupContainerACL(group, account, container string) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if v, ok := t.containerACLOverrides[containerACLKey(group, account, container)]; ok {
		return v
	}
	return "None"
}

func (t *transport) setContainerACL(group, account, container, level string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.containerACLOverrides[containerACLKey(group, account, container)] = level
}

func containerExists(account storageAccountFixture, name string) bool {
	for _, c := range account.BlobContainers {
		if c == name {
			return true
		}
	}
	return false
}

func readBody(req *http.Request) ([]byte, error) {
	return demoreplay.ReadRequestBody(req)
}

// handleRoleAssignments serves PUT/GET/DELETE under
// /subscriptions/{sub}/providers/Microsoft.Authorization/roleAssignments[/{name}].
func (t *transport) handleRoleAssignments(req *http.Request, subscription string, parts []string) (*http.Response, error) {
	scope := "/subscriptions/" + subscription
	switch len(parts) {
	case 0:
		if req.Method != http.MethodGet {
			return armErrorResponse(req, http.StatusMethodNotAllowed, "MethodNotAllowed",
				fmt.Sprintf("method %s not supported on roleAssignments", req.Method)), nil
		}
		filter := req.URL.Query().Get("$filter")
		principalFilter := parsePrincipalIDFilter(filter)
		assignments := t.activeAssignments(scope)
		resp := azapi.ListRoleAssignmentsResponse{}
		for _, a := range assignments {
			if principalFilter != "" && !strings.EqualFold(a.PrincipalID, principalFilter) {
				continue
			}
			resp.Value = append(resp.Value, t.assignmentToARM(scope, a))
		}
		return jsonResponse(req, resp), nil
	case 1:
		assignmentName := parts[0]
		switch req.Method {
		case http.MethodPut:
			body, err := readBody(req)
			if err != nil {
				return armErrorResponse(req, http.StatusBadRequest, "InvalidRequestBody", err.Error()), nil
			}
			var payload azapi.CreateRoleAssignmentRequest
			if err := json.Unmarshal(body, &payload); err != nil {
				return armErrorResponse(req, http.StatusBadRequest, "InvalidRequestBody", err.Error()), nil
			}
			if !isKnownPrincipal(payload.Properties.PrincipalID) {
				return armErrorResponse(req, http.StatusBadRequest, "PrincipalNotFound",
					fmt.Sprintf("principal %s not found in directory", payload.Properties.PrincipalID)), nil
			}
			defGUID := extractRoleDefinitionGUID(payload.Properties.RoleDefinitionID)
			if _, ok := roleDefinitionByGUID(defGUID); !ok {
				return armErrorResponse(req, http.StatusBadRequest, "RoleDefinitionDoesNotExist",
					fmt.Sprintf("role definition %s does not exist", payload.Properties.RoleDefinitionID)), nil
			}
			fixture := roleAssignmentFixture{
				Name:             assignmentName,
				PrincipalID:      payload.Properties.PrincipalID,
				RoleDefinitionID: defGUID,
				Scope:            scope,
			}
			t.storeAssignment(fixture)
			return jsonResponse(req, t.assignmentToARM(scope, fixture)), nil
		case http.MethodGet:
			fixture, ok := t.findAssignment(assignmentName, scope)
			if !ok {
				return armErrorResponse(req, http.StatusNotFound, "RoleAssignmentNotFound",
					fmt.Sprintf("role assignment %s not found", assignmentName)), nil
			}
			return jsonResponse(req, t.assignmentToARM(scope, fixture)), nil
		case http.MethodDelete:
			fixture, ok := t.findAssignment(assignmentName, scope)
			if !ok {
				return armErrorResponse(req, http.StatusNotFound, "RoleAssignmentNotFound",
					fmt.Sprintf("role assignment %s not found", assignmentName)), nil
			}
			t.deleteAssignment(assignmentName)
			return jsonResponse(req, t.assignmentToARM(scope, fixture)), nil
		}
		return armErrorResponse(req, http.StatusMethodNotAllowed, "MethodNotAllowed",
			fmt.Sprintf("method %s not supported on roleAssignment", req.Method)), nil
	}
	return armErrorResponse(req, http.StatusNotFound, "InvalidPath",
		fmt.Sprintf("unsupported roleAssignments path: %v", parts)), nil
}

// handleRoleDefinitions serves GET under
// /subscriptions/{sub}/providers/Microsoft.Authorization/roleDefinitions[?$filter=...].
func (t *transport) handleRoleDefinitions(req *http.Request, subscription string, parts []string) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return armErrorResponse(req, http.StatusMethodNotAllowed, "MethodNotAllowed",
			fmt.Sprintf("method %s not supported on roleDefinitions", req.Method)), nil
	}
	filter := req.URL.Query().Get("$filter")
	wantName := parseRoleNameFilter(filter)
	scope := "/subscriptions/" + subscription
	resp := azapi.ListRoleDefinitionsResponse{}
	for _, def := range demoRoleDefinitions {
		if wantName != "" && !strings.EqualFold(def.Name, wantName) {
			continue
		}
		resp.Value = append(resp.Value, azapi.RoleDefinition{
			ID:   scope + "/providers/Microsoft.Authorization/roleDefinitions/" + def.GUID,
			Name: def.GUID,
			Type: "Microsoft.Authorization/roleDefinitions",
			Properties: azapi.RoleDefinitionProperties{
				RoleName: def.Name,
				Type:     "BuiltInRole",
			},
		})
	}
	return jsonResponse(req, resp), nil
}

func (t *transport) activeAssignments(scope string) []roleAssignmentFixture {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]roleAssignmentFixture, 0, len(demoRoleAssignments)+len(t.createdAssignments))
	for _, a := range demoRoleAssignments {
		if t.deletedAssignments[a.Name] {
			continue
		}
		copy := a
		if copy.Scope == "" {
			copy.Scope = scope
		}
		out = append(out, copy)
	}
	for _, a := range t.createdAssignments {
		if t.deletedAssignments[a.Name] {
			continue
		}
		out = append(out, a)
	}
	return out
}

func (t *transport) findAssignment(name, scope string) (roleAssignmentFixture, bool) {
	for _, a := range t.activeAssignments(scope) {
		if a.Name == name {
			return a, true
		}
	}
	return roleAssignmentFixture{}, false
}

func (t *transport) storeAssignment(fixture roleAssignmentFixture) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.createdAssignments[fixture.Name] = fixture
	delete(t.deletedAssignments, fixture.Name)
}

func (t *transport) deleteAssignment(name string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.createdAssignments, name)
	t.deletedAssignments[name] = true
}

func (t *transport) assignmentToARM(scope string, fixture roleAssignmentFixture) azapi.RoleAssignment {
	target := scope
	if fixture.Scope != "" {
		target = fixture.Scope
	}
	return azapi.RoleAssignment{
		ID:   target + "/providers/Microsoft.Authorization/roleAssignments/" + fixture.Name,
		Name: fixture.Name,
		Type: "Microsoft.Authorization/roleAssignments",
		Properties: azapi.RoleAssignmentProperties{
			RoleDefinitionID: target + "/providers/Microsoft.Authorization/roleDefinitions/" + fixture.RoleDefinitionID,
			PrincipalID:      fixture.PrincipalID,
			Scope:            target,
		},
	}
}

func extractRoleDefinitionGUID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	idx := strings.LastIndex(id, "/")
	if idx < 0 {
		return id
	}
	return id[idx+1:]
}

func parseRoleNameFilter(filter string) string {
	return parseEqFilter(filter, "roleName")
}

func parsePrincipalIDFilter(filter string) string {
	return parseEqFilter(filter, "principalId")
}

// parseEqFilter pulls "value" out of a filter clause shaped like
// "<field> eq '<value>'". Returns "" when the clause does not match. Replay
// only needs the equality form.
func parseEqFilter(filter, field string) string {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return ""
	}
	prefix := field + " eq '"
	idx := strings.Index(filter, prefix)
	if idx < 0 {
		return ""
	}
	rest := filter[idx+len(prefix):]
	end := strings.Index(rest, "'")
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// handleGraph routes Microsoft Graph requests for the demo replay. The
// supported surface covers user inventory/lifecycle and application password
// credentials used by defensive validation payloads.
func (t *transport) handleGraph(req *http.Request, body []byte) (*http.Response, error) {
	path := strings.Trim(req.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] != "v1.0" {
		return graphErrorResponse(req, http.StatusNotFound, "InvalidGraphPath",
			"unsupported graph path: "+req.URL.Path), nil
	}
	parts = parts[1:]
	switch {
	case len(parts) == 1 && parts[0] == "applications":
		return t.handleGraphListApplications(req)
	case len(parts) == 2 && parts[0] == "applications":
		return t.handleGraphGetApplication(req, parts[1])
	case len(parts) == 3 && parts[0] == "applications" && parts[2] == "addPassword":
		return t.handleGraphAddPassword(req, parts[1], body)
	case len(parts) == 3 && parts[0] == "applications" && parts[2] == "removePassword":
		return t.handleGraphRemovePassword(req, parts[1], body)
	case len(parts) == 1 && parts[0] == "users" && req.Method == http.MethodGet:
		return t.handleGraphListUsers(req)
	case len(parts) == 1 && parts[0] == "users" && req.Method == http.MethodPost:
		return t.handleGraphCreateUser(req, body)
	case len(parts) == 2 && parts[0] == "users" && req.Method == http.MethodDelete:
		return t.handleGraphDeleteUser(req, parts[1])
	}
	return graphErrorResponse(req, http.StatusNotFound, "InvalidGraphPath",
		"unsupported graph path: "+req.URL.Path), nil
}

func (t *transport) handleGraphCreateUser(req *http.Request, body []byte) (*http.Response, error) {
	var user struct {
		AccountEnabled    bool   `json:"accountEnabled"`
		DisplayName       string `json:"displayName"`
		MailNickname      string `json:"mailNickname"`
		UserPrincipalName string `json:"userPrincipalName"`
	}
	_ = json.Unmarshal(body, &user)
	if strings.TrimSpace(user.UserPrincipalName) == "" {
		return graphErrorResponse(req, http.StatusBadRequest, "Request_BadRequest",
			"userPrincipalName required"), nil
	}
	t.addGraphUser(user.UserPrincipalName)
	return jsonResponse(req, map[string]any{
		"id":                "ctk-graph-user-001",
		"accountEnabled":    user.AccountEnabled,
		"displayName":       user.DisplayName,
		"mailNickname":      user.MailNickname,
		"userPrincipalName": user.UserPrincipalName,
	}), nil
}

func (t *transport) handleGraphListUsers(req *http.Request) (*http.Response, error) {
	users := make([]map[string]any, 0, len(demoGraphUsers))
	for _, user := range demoGraphUsers {
		users = append(users, buildGraphUser(user))
	}
	t.mu.Lock()
	for upn := range t.graphUsers {
		users = append(users, buildGraphUser(graphUserFixture{
			ID:                "ctk-created-" + upn,
			DisplayName:       upn,
			UserPrincipalName: upn,
			AccountEnabled:    true,
			CreatedDateTime:   "2026-05-06T00:00:00Z",
		}))
	}
	t.mu.Unlock()
	return jsonResponse(req, map[string]any{"value": users}), nil
}

func (t *transport) handleGraphDeleteUser(req *http.Request, idOrUPN string) (*http.Response, error) {
	// Lenient: removeGraphUser returns false if the user wasn't seen in this
	// session, but we still want to surface a 204 because demo flows might
	// run `del` against a baseline UPN.
	t.removeGraphUser(idOrUPN)
	resp := demoreplay.Response(req, http.StatusNoContent, "application/json", nil)
	resp.Header.Set("request-id", "req-replay-graph")
	return resp, nil
}

func (t *transport) handleGraphListApplications(req *http.Request) (*http.Response, error) {
	filter := req.URL.Query().Get("$filter")
	wantAppID := extractGraphFilterValue(filter, "appId eq '")
	apps := make([]map[string]any, 0)
	for _, app := range demoGraphApplications {
		if wantAppID != "" && app.AppID != wantAppID {
			continue
		}
		apps = append(apps, t.buildGraphApplication(app))
	}
	return jsonResponse(req, map[string]any{"value": apps}), nil
}

func (t *transport) handleGraphGetApplication(req *http.Request, id string) (*http.Response, error) {
	app, ok := findGraphApplicationByID(id)
	if !ok {
		return graphErrorResponse(req, http.StatusNotFound, "Request_ResourceNotFound",
			"application not found"), nil
	}
	return jsonResponse(req, t.buildGraphApplication(app)), nil
}

func buildGraphUser(user graphUserFixture) map[string]any {
	out := map[string]any{
		"id":                user.ID,
		"accountEnabled":    user.AccountEnabled,
		"displayName":       user.DisplayName,
		"userPrincipalName": user.UserPrincipalName,
		"createdDateTime":   user.CreatedDateTime,
	}
	if strings.TrimSpace(user.LastSignInDateTime) != "" {
		out["signInActivity"] = map[string]any{
			"lastSignInDateTime": user.LastSignInDateTime,
		}
	}
	return out
}

func (t *transport) handleGraphAddPassword(req *http.Request, id string, body []byte) (*http.Response, error) {
	app, ok := findGraphApplicationByID(id)
	if !ok {
		return graphErrorResponse(req, http.StatusNotFound, "Request_ResourceNotFound",
			"application not found"), nil
	}
	var payload struct {
		PasswordCredential struct {
			DisplayName string `json:"displayName"`
		} `json:"passwordCredential"`
	}
	_ = json.Unmarshal(body, &payload)
	pc := t.mintAzureAppPassword(app.ID, payload.PasswordCredential.DisplayName)
	return jsonResponse(req, map[string]any{
		"keyId":         pc.KeyID,
		"displayName":   pc.DisplayName,
		"startDateTime": pc.StartDateTime,
		"endDateTime":   pc.EndDateTime,
		"secretText":    pc.SecretText,
		"hint":          pc.Hint,
	}), nil
}

func (t *transport) handleGraphRemovePassword(req *http.Request, id string, body []byte) (*http.Response, error) {
	app, ok := findGraphApplicationByID(id)
	if !ok {
		return graphErrorResponse(req, http.StatusNotFound, "Request_ResourceNotFound",
			"application not found"), nil
	}
	var payload struct {
		KeyID string `json:"keyId"`
	}
	_ = json.Unmarshal(body, &payload)
	if !t.deleteAzureAppPassword(app.ID, payload.KeyID) {
		return graphErrorResponse(req, http.StatusNotFound, "Request_ResourceNotFound",
			"password credential not found"), nil
	}
	resp := demoreplay.Response(req, http.StatusNoContent, "application/json", nil)
	resp.Header.Set("request-id", "req-replay-graph")
	return resp, nil
}

func (t *transport) buildGraphApplication(app graphApplicationFixture) map[string]any {
	t.mu.Lock()
	creds := append([]graphPasswordFixture(nil), t.appPasswords[app.ID]...)
	t.mu.Unlock()
	out := []map[string]any{}
	for _, c := range creds {
		out = append(out, map[string]any{
			"keyId":         c.KeyID,
			"displayName":   c.DisplayName,
			"startDateTime": c.StartDateTime,
			"endDateTime":   c.EndDateTime,
		})
	}
	return map[string]any{
		"id":                  app.ID,
		"appId":               app.AppID,
		"displayName":         app.DisplayName,
		"passwordCredentials": out,
	}
}

// extractGraphFilterValue parses simple `eq '...'` clauses out of a Graph
// $filter query parameter. Returns "" if the prefix isn't found.
func extractGraphFilterValue(filter, prefix string) string {
	idx := strings.Index(filter, prefix)
	if idx < 0 {
		return ""
	}
	rest := filter[idx+len(prefix):]
	end := strings.Index(rest, "'")
	if end < 0 {
		return ""
	}
	return rest[:end]
}
