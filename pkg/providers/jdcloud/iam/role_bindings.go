package iam

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/jdcloud/api"
	"github.com/404tk/cloudtoolkit/pkg/schema"
)

// ListRoleBindings returns the managed policies attached to the named sub user.
// JDCloud IAM is global; the region argument is left empty so the signer falls
// back to the `jdcloud-api` scope used by the other sub-user actions.
//
// The action path follows the existing :attachSubUserPolicy / :detachSubUserPolicy
// pattern. If JDCloud renames the describe action upstream, both this driver and
// the demo replay handler in pkg/providers/jdcloud/replay need updating in lockstep.
func (d *Driver) ListRoleBindings(ctx context.Context, userName string) ([]schema.RoleBinding, error) {
	if d.Client == nil {
		return nil, fmt.Errorf("jdcloud iam: nil api client")
	}
	userName = strings.TrimSpace(userName)
	if userName == "" {
		return nil, fmt.Errorf("jdcloud iam: principal (sub user name) required for list")
	}
	var resp api.DescribeAttachedPoliciesResponse
	if err := d.Client.DoJSON(ctx, api.Request{
		Service: "iam",
		Region:  "",
		Method:  http.MethodGet,
		Version: "v1",
		Path:    "/subUser/" + userName + ":describeAttachedPolicies",
	}, &resp); err != nil {
		return nil, err
	}
	bindings := make([]schema.RoleBinding, 0, len(resp.Result.Policies))
	for _, p := range resp.Result.Policies {
		bindings = append(bindings, schema.RoleBinding{
			Principal: userName,
			Role:      p.PolicyName,
			Scope:     p.PolicyType,
		})
	}
	return bindings, nil
}

// AttachPolicy binds policyName to userName. Reuses the same payload shape that
// the existing iam-user-check `add` flow already exercises.
func (d *Driver) AttachPolicy(ctx context.Context, userName, policyName string) error {
	if d.Client == nil {
		return fmt.Errorf("jdcloud iam: nil api client")
	}
	userName = strings.TrimSpace(userName)
	policyName = strings.TrimSpace(policyName)
	if userName == "" || policyName == "" {
		return fmt.Errorf("jdcloud iam: principal and role required")
	}
	body, err := json.Marshal(api.AttachSubUserPolicyRequest{SubUser: userName, PolicyName: policyName})
	if err != nil {
		return err
	}
	var resp api.AttachSubUserPolicyResponse
	return d.Client.DoJSON(ctx, api.Request{
		Service: "iam",
		Region:  "",
		Method:  http.MethodPost,
		Version: "v1",
		Path:    "/subUser/" + userName + ":attachSubUserPolicy",
		Body:    body,
	}, &resp)
}

// DetachPolicy removes policyName from userName. Mirrors `userdel`: JDCloud's
// detach action expects HTTP DELETE with policyName in the query string.
func (d *Driver) DetachPolicy(ctx context.Context, userName, policyName string) error {
	if d.Client == nil {
		return fmt.Errorf("jdcloud iam: nil api client")
	}
	userName = strings.TrimSpace(userName)
	policyName = strings.TrimSpace(policyName)
	if userName == "" || policyName == "" {
		return fmt.Errorf("jdcloud iam: principal and role required")
	}
	query := url.Values{}
	query.Set("policyName", policyName)
	var resp api.DetachSubUserPolicyResponse
	return d.Client.DoJSON(ctx, api.Request{
		Service: "iam",
		Region:  "",
		Method:  http.MethodDelete,
		Version: "v1",
		Path:    "/subUser/" + userName + ":detachSubUserPolicy",
		Query:   query,
	}, &resp)
}
