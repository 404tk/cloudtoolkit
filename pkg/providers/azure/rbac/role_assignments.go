package rbac

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	azapi "github.com/404tk/cloudtoolkit/pkg/providers/azure/api"
)

// Driver wraps an authenticated ARM client for the
// Microsoft.Authorization/roleAssignments and roleDefinitions resources.
type Driver struct {
	Client          *azapi.Client
	SubscriptionIDs []string
}

// DefaultScope returns the canonical subscription-level scope for the first
// known subscription, or an empty string if no subscription is configured.
func (d *Driver) DefaultScope() string {
	if d == nil || len(d.SubscriptionIDs) == 0 || d.SubscriptionIDs[0] == "" {
		return ""
	}
	return "/subscriptions/" + d.SubscriptionIDs[0]
}

// List returns role assignments under the supplied scope. When principalID is
// non-empty it is forwarded as `$filter=principalId eq '...'`.
func (d *Driver) List(ctx context.Context, scope, principalID string) ([]azapi.RoleAssignment, error) {
	if d == nil || d.Client == nil {
		return nil, fmt.Errorf("azure rbac: nil client")
	}
	scope = normalizeScope(scope)
	query := url.Values{"api-version": {azapi.AuthorizationAPIVersion}}
	if principalID != "" {
		query.Set("$filter", fmt.Sprintf("principalId eq '%s'", principalID))
	}
	pager := azapi.NewPager[azapi.RoleAssignment](d.Client, azapi.Request{
		Method:     http.MethodGet,
		Path:       scope + "/providers/Microsoft.Authorization/roleAssignments",
		Query:      query,
		Idempotent: true,
	})
	return pager.All(ctx)
}

// Create binds principalID to the role identified by roleName at scope. The
// role name is resolved to a roleDefinition GUID via List on roleDefinitions.
func (d *Driver) Create(ctx context.Context, scope, principalID, roleName string) (azapi.RoleAssignment, error) {
	if d == nil || d.Client == nil {
		return azapi.RoleAssignment{}, fmt.Errorf("azure rbac: nil client")
	}
	scope = normalizeScope(scope)
	if principalID == "" {
		return azapi.RoleAssignment{}, fmt.Errorf("azure rbac: principalId required")
	}
	if roleName == "" {
		return azapi.RoleAssignment{}, fmt.Errorf("azure rbac: role name required")
	}
	roleDefID, err := d.lookupRoleDefinitionID(ctx, scope, roleName)
	if err != nil {
		return azapi.RoleAssignment{}, err
	}
	assignmentName, err := newAssignmentName()
	if err != nil {
		return azapi.RoleAssignment{}, err
	}
	body, err := json.Marshal(azapi.CreateRoleAssignmentRequest{
		Properties: azapi.RoleAssignmentProperties{
			RoleDefinitionID: roleDefID,
			PrincipalID:      principalID,
		},
	})
	if err != nil {
		return azapi.RoleAssignment{}, err
	}
	var assignment azapi.RoleAssignment
	if err := d.Client.Do(ctx, azapi.Request{
		Method: http.MethodPut,
		Path:   scope + "/providers/Microsoft.Authorization/roleAssignments/" + assignmentName,
		Query:  url.Values{"api-version": {azapi.AuthorizationAPIVersion}},
		Body:   body,
	}, &assignment); err != nil {
		return azapi.RoleAssignment{}, err
	}
	return assignment, nil
}

// Delete removes a role assignment. Either assignmentName (GUID) or the
// (principalID, roleName) tuple may be supplied; when principal/role are given
// the driver lists assignments at scope to resolve the GUID.
func (d *Driver) Delete(ctx context.Context, scope, assignmentName, principalID, roleName string) (string, error) {
	if d == nil || d.Client == nil {
		return "", fmt.Errorf("azure rbac: nil client")
	}
	scope = normalizeScope(scope)
	resolved := strings.TrimSpace(assignmentName)
	if resolved == "" {
		if principalID == "" || roleName == "" {
			return "", fmt.Errorf("azure rbac: delete requires either assignmentName or (principalId, roleName)")
		}
		roleDefID, err := d.lookupRoleDefinitionID(ctx, scope, roleName)
		if err != nil {
			return "", err
		}
		assignments, err := d.List(ctx, scope, principalID)
		if err != nil {
			return "", err
		}
		for _, a := range assignments {
			if strings.EqualFold(a.Properties.RoleDefinitionID, roleDefID) {
				resolved = a.Name
				break
			}
		}
		if resolved == "" {
			return "", fmt.Errorf("azure rbac: no role assignment for principal %s with role %s at %s", principalID, roleName, scope)
		}
	}
	if err := d.Client.Do(ctx, azapi.Request{
		Method: http.MethodDelete,
		Path:   scope + "/providers/Microsoft.Authorization/roleAssignments/" + resolved,
		Query:  url.Values{"api-version": {azapi.AuthorizationAPIVersion}},
	}, nil); err != nil {
		return "", err
	}
	return resolved, nil
}

// lookupRoleDefinitionID resolves a roleName like "Reader" / "Owner" to the
// fully-qualified roleDefinitionId required by role assignments.
func (d *Driver) lookupRoleDefinitionID(ctx context.Context, scope, roleName string) (string, error) {
	query := url.Values{
		"api-version": {azapi.AuthorizationAPIVersion},
		"$filter":     {fmt.Sprintf("roleName eq '%s'", roleName)},
	}
	pager := azapi.NewPager[azapi.RoleDefinition](d.Client, azapi.Request{
		Method:     http.MethodGet,
		Path:       scope + "/providers/Microsoft.Authorization/roleDefinitions",
		Query:      query,
		Idempotent: true,
	})
	defs, err := pager.All(ctx)
	if err != nil {
		return "", err
	}
	for _, def := range defs {
		if strings.EqualFold(def.Properties.RoleName, roleName) && def.ID != "" {
			return def.ID, nil
		}
	}
	return "", fmt.Errorf("azure rbac: role %q not found at scope %s", roleName, scope)
}

func normalizeScope(scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return ""
	}
	if !strings.HasPrefix(scope, "/") {
		scope = "/" + scope
	}
	return strings.TrimRight(scope, "/")
}

// newAssignmentName returns a random RFC4122-shaped GUID used as the
// roleAssignment resource name. Azure requires the resource name to be a
// fresh GUID for every create.
func newAssignmentName() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}
