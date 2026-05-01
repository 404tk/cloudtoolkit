package iam

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// ListRoleBindings returns the policies currently attached to the named CAM
// sub-user. Tencent CAM identifies users by numeric Uin internally, so the
// driver resolves the supplied name via GetUser before paginating policies.
func (d *Driver) ListRoleBindings(ctx context.Context, userName string) ([]schema.RoleBinding, error) {
	userName = strings.TrimSpace(userName)
	if userName == "" {
		return nil, fmt.Errorf("tencent iam: principal (CAM user name) required for list")
	}
	client := d.newClient()
	userResp, err := client.GetUser(ctx, userName)
	if err != nil {
		return nil, err
	}
	uin := derefUint64(userResp.Response.Uin)
	if uin == 0 {
		return nil, fmt.Errorf("tencent iam: cannot resolve uin for user %s", userName)
	}
	bindings := make([]schema.RoleBinding, 0)
	page := uint64(1)
	for {
		resp, err := client.ListAttachedUserAllPolicies(ctx, uin, page, 50, 0)
		if err != nil {
			return nil, err
		}
		for _, p := range resp.Response.PolicyList {
			bindings = append(bindings, schema.RoleBinding{
				Principal:    userName,
				Role:         derefString(p.PolicyName),
				Scope:        derefString(p.PolicyID),
				AssignmentID: derefString(p.PolicyID),
			})
		}
		total := derefUint64(resp.Response.TotalNum)
		if uint64(len(bindings)) >= total || len(resp.Response.PolicyList) == 0 {
			break
		}
		page++
	}
	return bindings, nil
}

// AttachPolicy binds policyID to the named CAM user.
func (d *Driver) AttachPolicy(ctx context.Context, userName string, policyID uint64) error {
	userName = strings.TrimSpace(userName)
	if userName == "" || policyID == 0 {
		return fmt.Errorf("tencent iam: principal and policyID required")
	}
	client := d.newClient()
	userResp, err := client.GetUser(ctx, userName)
	if err != nil {
		return err
	}
	uin := derefUint64(userResp.Response.Uin)
	if uin == 0 {
		return fmt.Errorf("tencent iam: cannot resolve uin for user %s", userName)
	}
	_, err = client.AttachUserPolicy(ctx, uin, policyID)
	return err
}

// DetachPolicy removes policyID from the named CAM user.
func (d *Driver) DetachPolicy(ctx context.Context, userName string, policyID uint64) error {
	userName = strings.TrimSpace(userName)
	if userName == "" || policyID == 0 {
		return fmt.Errorf("tencent iam: principal and policyID required")
	}
	client := d.newClient()
	userResp, err := client.GetUser(ctx, userName)
	if err != nil {
		return err
	}
	uin := derefUint64(userResp.Response.Uin)
	if uin == 0 {
		return fmt.Errorf("tencent iam: cannot resolve uin for user %s", userName)
	}
	_, err = client.DetachUserPolicy(ctx, uin, policyID)
	return err
}

// ResolvePolicyID accepts a numeric string ("200001") or a friendly name
// ("AdministratorAccess") and returns the corresponding CAM policy ID. The
// friendly-name mapping is intentionally narrow — call sites that need a
// long-tail policy should pass the numeric ID directly.
func ResolvePolicyID(role string) (uint64, error) {
	role = strings.TrimSpace(role)
	if role == "" {
		return 0, fmt.Errorf("tencent iam: empty policy identifier")
	}
	if id, err := strconv.ParseUint(role, 10, 64); err == nil {
		return id, nil
	}
	switch strings.ToLower(role) {
	case "administratoraccess", "qcloudresourcefullaccess":
		return 1, nil
	}
	return 0, fmt.Errorf("tencent iam: cannot resolve policy %q to a numeric ID; pass the integer policyID instead", role)
}
