package iam

import (
	"context"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/alibaba/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// ListRoleBindings returns the policies attached to the supplied RAM user.
// Alibaba RAM does not expose an account-wide enumeration of policy
// attachments, so userName is required.
func (d *Driver) ListRoleBindings(ctx context.Context, userName string) ([]schema.RoleBinding, error) {
	if d == nil {
		return nil, fmt.Errorf("alibaba iam: nil driver")
	}
	userName = strings.TrimSpace(userName)
	if userName == "" {
		return nil, fmt.Errorf("alibaba iam: principal (RAM user name) required for list")
	}
	client := d.newClient()
	region := api.NormalizeRegion(d.Region)
	resp, err := client.ListRAMPoliciesForUser(ctx, region, userName)
	if err != nil {
		return nil, err
	}
	bindings := make([]schema.RoleBinding, 0, len(resp.Policies.Policy))
	for _, p := range resp.Policies.Policy {
		bindings = append(bindings, schema.RoleBinding{
			Principal: userName,
			Role:      p.PolicyName,
			Scope:     p.PolicyType,
		})
	}
	return bindings, nil
}

// AttachPolicyToUser binds the named RAM policy to userName. policyType is
// `System` (built-in) or `Custom` (account-defined); empty defaults to System.
func (d *Driver) AttachPolicyToUser(ctx context.Context, userName, policyName, policyType string) error {
	if d == nil {
		return fmt.Errorf("alibaba iam: nil driver")
	}
	userName = strings.TrimSpace(userName)
	policyName = strings.TrimSpace(policyName)
	if userName == "" || policyName == "" {
		return fmt.Errorf("alibaba iam: principal and role required")
	}
	policyType = normalizePolicyType(policyType)
	client := d.newClient()
	region := api.NormalizeRegion(d.Region)
	_, err := client.AttachRAMPolicyToUser(ctx, region, userName, policyName, policyType)
	return err
}

// DetachPolicyFromUser removes the named RAM policy from userName.
func (d *Driver) DetachPolicyFromUser(ctx context.Context, userName, policyName, policyType string) error {
	if d == nil {
		return fmt.Errorf("alibaba iam: nil driver")
	}
	userName = strings.TrimSpace(userName)
	policyName = strings.TrimSpace(policyName)
	if userName == "" || policyName == "" {
		return fmt.Errorf("alibaba iam: principal and role required")
	}
	policyType = normalizePolicyType(policyType)
	client := d.newClient()
	region := api.NormalizeRegion(d.Region)
	_, err := client.DetachRAMPolicyFromUser(ctx, region, userName, policyName, policyType)
	return err
}

func normalizePolicyType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "System"
	}
	if strings.EqualFold(value, "system") {
		return "System"
	}
	if strings.EqualFold(value, "custom") {
		return "Custom"
	}
	return value
}
