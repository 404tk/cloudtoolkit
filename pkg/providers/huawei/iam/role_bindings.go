package iam

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/huawei/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// ListRoleBindings enumerates the keystone groups that user is a member of.
// Huawei IAM has no "policy attached to user" concept — policies attach to
// groups and users join groups, so the role-binding capability surfaces group
// membership as the closest CSPM-relevant analogue.
func (d *Driver) ListRoleBindings(ctx context.Context, userName string) ([]schema.RoleBinding, error) {
	userName = strings.TrimSpace(userName)
	if userName == "" {
		return nil, fmt.Errorf("huawei iam: principal (user name) required for list")
	}
	region, err := d.requestRegion()
	if err != nil {
		return nil, err
	}
	userID, err := d.lookupUserID(ctx, userName)
	if err != nil {
		return nil, err
	}
	var resp api.ListGroupsForUserResponse
	if err := d.client().DoJSON(ctx, api.Request{
		Service:    "iam",
		Region:     region,
		Intl:       d.Cred.Intl,
		Method:     http.MethodGet,
		Path:       fmt.Sprintf("/v3/users/%s/groups", userID),
		Idempotent: true,
		Headers:    d.domainHeaders(ctx),
	}, &resp); err != nil {
		return nil, err
	}
	bindings := make([]schema.RoleBinding, 0, len(resp.Groups))
	for _, g := range resp.Groups {
		bindings = append(bindings, schema.RoleBinding{
			Principal:    userName,
			Role:         g.Name,
			Scope:        "domain",
			AssignmentID: g.ID,
		})
	}
	return bindings, nil
}

// AttachGroup adds user to the named keystone group.
func (d *Driver) AttachGroup(ctx context.Context, userName, groupName string) error {
	groupID, userID, region, err := d.resolveBinding(ctx, userName, groupName)
	if err != nil {
		return err
	}
	return d.client().DoJSON(ctx, api.Request{
		Service: "iam",
		Region:  region,
		Intl:    d.Cred.Intl,
		Method:  http.MethodPut,
		Path:    fmt.Sprintf("/v3/groups/%s/users/%s", groupID, userID),
		Headers: d.domainHeaders(ctx),
	}, nil)
}

// DetachGroup removes user from the named keystone group.
func (d *Driver) DetachGroup(ctx context.Context, userName, groupName string) error {
	groupID, userID, region, err := d.resolveBinding(ctx, userName, groupName)
	if err != nil {
		return err
	}
	return d.client().DoJSON(ctx, api.Request{
		Service: "iam",
		Region:  region,
		Intl:    d.Cred.Intl,
		Method:  http.MethodDelete,
		Path:    fmt.Sprintf("/v3/groups/%s/users/%s", groupID, userID),
		Headers: d.domainHeaders(ctx),
	}, nil)
}

func (d *Driver) resolveBinding(ctx context.Context, userName, groupName string) (string, string, string, error) {
	userName = strings.TrimSpace(userName)
	groupName = strings.TrimSpace(groupName)
	if userName == "" || groupName == "" {
		return "", "", "", fmt.Errorf("huawei iam: principal and role (group name) required")
	}
	region, err := d.requestRegion()
	if err != nil {
		return "", "", "", err
	}
	userID, err := d.lookupUserID(ctx, userName)
	if err != nil {
		return "", "", "", err
	}
	groupID, err := d.lookupGroupID(ctx, groupName)
	if err != nil {
		return "", "", "", err
	}
	return groupID, userID, region, nil
}

func (d *Driver) lookupUserID(ctx context.Context, userName string) (string, error) {
	region, err := d.requestRegion()
	if err != nil {
		return "", err
	}
	var resp api.ListUsersV5Response
	if err := d.client().DoJSON(ctx, api.Request{
		Service:    "iam",
		Region:     region,
		Intl:       d.Cred.Intl,
		Method:     http.MethodGet,
		Path:       "/v5/users",
		Idempotent: true,
	}, &resp); err != nil {
		return "", err
	}
	for _, u := range resp.Users {
		if u.UserName == userName {
			return u.UserID, nil
		}
	}
	return "", fmt.Errorf("huawei iam: user %q not found", userName)
}

func (d *Driver) lookupGroupID(ctx context.Context, groupName string) (string, error) {
	region, err := d.requestRegion()
	if err != nil {
		return "", err
	}
	var resp api.ListGroupsResponse
	if err := d.client().DoJSON(ctx, api.Request{
		Service:    "iam",
		Region:     region,
		Intl:       d.Cred.Intl,
		Method:     http.MethodGet,
		Path:       "/v3/groups",
		Idempotent: true,
		Headers:    d.domainHeaders(ctx),
	}, &resp); err != nil {
		return "", err
	}
	for _, g := range resp.Groups {
		if g.Name == groupName {
			return g.ID, nil
		}
	}
	return "", fmt.Errorf("huawei iam: group %q not found", groupName)
}
