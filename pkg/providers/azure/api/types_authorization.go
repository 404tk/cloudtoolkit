package api

const AuthorizationAPIVersion = "2022-04-01"

// RoleAssignment represents a Microsoft.Authorization/roleAssignments resource
// in Azure Resource Manager.
type RoleAssignment struct {
	ID         string                   `json:"id"`
	Name       string                   `json:"name"`
	Type       string                   `json:"type"`
	Properties RoleAssignmentProperties `json:"properties"`
}

type RoleAssignmentProperties struct {
	RoleDefinitionID string `json:"roleDefinitionId"`
	PrincipalID      string `json:"principalId"`
	PrincipalType    string `json:"principalType,omitempty"`
	Scope            string `json:"scope,omitempty"`
}

type ListRoleAssignmentsResponse struct {
	Value    []RoleAssignment `json:"value"`
	NextLink string           `json:"nextLink"`
}

// CreateRoleAssignmentRequest is the body of PUT
// /{scope}/providers/Microsoft.Authorization/roleAssignments/{name}.
type CreateRoleAssignmentRequest struct {
	Properties RoleAssignmentProperties `json:"properties"`
}

// RoleDefinition represents a Microsoft.Authorization/roleDefinitions resource.
type RoleDefinition struct {
	ID         string                   `json:"id"`
	Name       string                   `json:"name"`
	Type       string                   `json:"type"`
	Properties RoleDefinitionProperties `json:"properties"`
}

type RoleDefinitionProperties struct {
	RoleName    string `json:"roleName"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
}

type ListRoleDefinitionsResponse struct {
	Value    []RoleDefinition `json:"value"`
	NextLink string           `json:"nextLink"`
}
