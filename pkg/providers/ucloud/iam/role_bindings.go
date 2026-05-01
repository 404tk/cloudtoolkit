package iam

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/ucloud/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// ListRoleBindings returns the policies attached to userName.
//
// UCloud's IAM `ListPoliciesForUser` action surfaces policy URNs (e.g.
// `ucs:iam::ucs:policy/AdministratorAccess`) along with whether the binding is
// account-wide (`Unspecified`) or scoped to a project (`Specified` + ProjectID).
func (d *Driver) ListRoleBindings(ctx context.Context, userName string) ([]schema.RoleBinding, error) {
	userName = strings.TrimSpace(userName)
	if userName == "" {
		return nil, fmt.Errorf("ucloud iam: principal (user name) required for list")
	}
	client := d.client()
	bindings := make([]schema.RoleBinding, 0)
	for offset := 0; ; offset += listUsersPageSize {
		var resp api.IAMListPoliciesForUserResponse
		err := client.Do(ctx, api.Request{
			Action: "ListPoliciesForUser",
			Params: map[string]any{
				"UserName": userName,
				"Limit":    strconv.Itoa(listUsersPageSize),
				"Offset":   strconv.Itoa(offset),
			},
			Idempotent: true,
		}, &resp)
		if err != nil {
			return nil, err
		}
		for _, p := range resp.Policies {
			scope := strings.TrimSpace(p.Scope)
			if scope == "" {
				if p.ProjectID != "" {
					scope = "Specified"
				} else {
					scope = "Unspecified"
				}
			}
			role := p.PolicyName
			if role == "" {
				role = p.PolicyURN
			}
			bindings = append(bindings, schema.RoleBinding{
				Principal:    userName,
				Role:         role,
				Scope:        scope,
				AssignmentID: p.PolicyURN,
			})
		}
		if len(resp.Policies) == 0 || len(resp.Policies) < listUsersPageSize ||
			(resp.TotalCount > 0 && offset+len(resp.Policies) >= resp.TotalCount) {
			break
		}
	}
	return bindings, nil
}

// AttachPolicy binds policyURN to userName. scope is "Specified" (project-
// scoped, requires Driver.ProjectID) or "Unspecified" (account-wide); empty
// defaults to "Unspecified".
func (d *Driver) AttachPolicy(ctx context.Context, userName, policyURN, scope string) error {
	userName = strings.TrimSpace(userName)
	policyURN = strings.TrimSpace(policyURN)
	if userName == "" || policyURN == "" {
		return fmt.Errorf("ucloud iam: principal and role required")
	}
	scope = normalizeUCloudScope(scope)
	params := map[string]any{
		"UserName":   userName,
		"PolicyURNs": []string{policyURN},
		"Scope":      scope,
	}
	if scope == "Specified" {
		if strings.TrimSpace(d.ProjectID) == "" {
			return fmt.Errorf("ucloud iam: scope=Specified requires ProjectID on driver")
		}
		params["ProjectID"] = d.ProjectID
	}
	var resp api.IAMAttachPoliciesToUserResponse
	return d.client().Do(ctx, api.Request{Action: "AttachPoliciesToUser", Params: params}, &resp)
}

// DetachPolicy removes policyURN from userName.
func (d *Driver) DetachPolicy(ctx context.Context, userName, policyURN, scope string) error {
	userName = strings.TrimSpace(userName)
	policyURN = strings.TrimSpace(policyURN)
	if userName == "" || policyURN == "" {
		return fmt.Errorf("ucloud iam: principal and role required")
	}
	scope = normalizeUCloudScope(scope)
	params := map[string]any{
		"UserName":   userName,
		"PolicyURNs": []string{policyURN},
		"Scope":      scope,
	}
	if scope == "Specified" && strings.TrimSpace(d.ProjectID) != "" {
		params["ProjectID"] = d.ProjectID
	}
	var resp api.IAMDetachPoliciesFromUserResponse
	return d.client().Do(ctx, api.Request{Action: "DetachPoliciesFromUser", Params: params}, &resp)
}

// ResolvePolicyURN expands a bare policy name ("AdministratorAccess") into the
// fully-qualified UCloud policy URN. Already-qualified URNs are returned as-is.
func ResolvePolicyURN(role string) string {
	role = strings.TrimSpace(role)
	if role == "" {
		return ""
	}
	if strings.HasPrefix(role, "ucs:") {
		return role
	}
	return "ucs:iam::ucs:policy/" + role
}

func normalizeUCloudScope(value string) string {
	value = strings.TrimSpace(value)
	switch strings.ToLower(value) {
	case "", "unspecified", "global", "account":
		return "Unspecified"
	case "specified", "project", "scoped":
		return "Specified"
	}
	return value
}
