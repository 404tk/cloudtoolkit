package iam

import (
	"context"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// ListRoleBindings returns the policies attached to userName.
func (d *Driver) ListRoleBindings(ctx context.Context, userName string) ([]schema.RoleBinding, error) {
	userName = strings.TrimSpace(userName)
	if userName == "" {
		return nil, fmt.Errorf("volcengine iam: principal (IAM user name) required for list")
	}
	client, err := d.requireClient()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListAttachedUserPolicies(ctx, d.requestRegion(), userName)
	if err != nil {
		return nil, err
	}
	bindings := make([]schema.RoleBinding, 0, len(resp.Result.AttachedPolicyMetadata))
	for _, p := range resp.Result.AttachedPolicyMetadata {
		bindings = append(bindings, schema.RoleBinding{
			Principal:    userName,
			Role:         p.PolicyName,
			Scope:        p.PolicyType,
			AssignmentID: p.PolicyTrn,
		})
	}
	return bindings, nil
}

// AttachPolicy binds the named policy to userName. policyType defaults to System.
func (d *Driver) AttachPolicy(ctx context.Context, userName, policyName, policyType string) error {
	userName = strings.TrimSpace(userName)
	policyName = strings.TrimSpace(policyName)
	if userName == "" || policyName == "" {
		return fmt.Errorf("volcengine iam: principal and role required")
	}
	policyType = normalizePolicyType(policyType)
	client, err := d.requireClient()
	if err != nil {
		return err
	}
	_, err = client.AttachUserPolicy(ctx, d.requestRegion(), userName, policyName, policyType)
	return err
}

// DetachPolicy removes the named policy from userName.
func (d *Driver) DetachPolicy(ctx context.Context, userName, policyName, policyType string) error {
	userName = strings.TrimSpace(userName)
	policyName = strings.TrimSpace(policyName)
	if userName == "" || policyName == "" {
		return fmt.Errorf("volcengine iam: principal and role required")
	}
	policyType = normalizePolicyType(policyType)
	client, err := d.requireClient()
	if err != nil {
		return err
	}
	_, err = client.DetachUserPolicy(ctx, d.requestRegion(), userName, policyName, policyType)
	return err
}

func normalizePolicyType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return systemPolicyType
	}
	if strings.EqualFold(value, "system") {
		return "System"
	}
	if strings.EqualFold(value, "custom") {
		return "Custom"
	}
	return value
}
