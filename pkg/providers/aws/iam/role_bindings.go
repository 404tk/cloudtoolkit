package iam

import (
	"context"
	"fmt"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/aws/api"
	"github.com/404tk/cloudtoolkit/pkg/runtime/paginate"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// ListRoleBindings returns the managed policies attached to userName.
// AWS IAM has no account-wide enumeration of attachments; userName is required.
func (d *Driver) ListRoleBindings(ctx context.Context, userName string) ([]schema.RoleBinding, error) {
	userName = strings.TrimSpace(userName)
	if userName == "" {
		return nil, fmt.Errorf("aws iam: principal (IAM user name) required for list")
	}
	client, err := d.requireClient()
	if err != nil {
		return nil, err
	}
	region := d.requestRegion()
	policies, err := paginate.Fetch[api.AttachedUserPolicy, string](ctx, func(ctx context.Context, marker string) (paginate.Page[api.AttachedUserPolicy, string], error) {
		resp, err := client.ListAttachedUserPolicies(ctx, region, userName, marker)
		if err != nil {
			return paginate.Page[api.AttachedUserPolicy, string]{}, err
		}
		return paginate.Page[api.AttachedUserPolicy, string]{
			Items: resp.Policies,
			Next:  resp.Marker,
			Done:  !resp.IsTruncated || strings.TrimSpace(resp.Marker) == "",
		}, nil
	})
	if err != nil {
		return nil, err
	}
	bindings := make([]schema.RoleBinding, 0, len(policies))
	for _, p := range policies {
		bindings = append(bindings, schema.RoleBinding{
			Principal:    userName,
			Role:         p.PolicyName,
			Scope:        p.PolicyArn,
			AssignmentID: p.PolicyArn,
		})
	}
	return bindings, nil
}

// AttachPolicy binds policyARN to userName.
func (d *Driver) AttachPolicy(ctx context.Context, userName, policyARN string) error {
	userName = strings.TrimSpace(userName)
	policyARN = strings.TrimSpace(policyARN)
	if userName == "" || policyARN == "" {
		return fmt.Errorf("aws iam: principal and role (policy ARN) required")
	}
	client, err := d.requireClient()
	if err != nil {
		return err
	}
	return client.AttachUserPolicy(ctx, d.requestRegion(), userName, policyARN)
}

// DetachPolicy removes policyARN from userName.
func (d *Driver) DetachPolicy(ctx context.Context, userName, policyARN string) error {
	userName = strings.TrimSpace(userName)
	policyARN = strings.TrimSpace(policyARN)
	if userName == "" || policyARN == "" {
		return fmt.Errorf("aws iam: principal and role (policy ARN) required")
	}
	client, err := d.requireClient()
	if err != nil {
		return err
	}
	return client.DetachUserPolicy(ctx, d.requestRegion(), userName, policyARN)
}

// ResolvePolicyARN expands a bare policy name (e.g. "AdministratorAccess") into
// its AWS-managed ARN. ARNs are returned untouched. Used so callers can pass
// either form to the role-binding-check payload.
func ResolvePolicyARN(role string) string {
	role = strings.TrimSpace(role)
	if role == "" {
		return ""
	}
	if strings.HasPrefix(role, "arn:") {
		return role
	}
	return "arn:aws:iam::aws:policy/" + role
}
