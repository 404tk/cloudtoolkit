package iam

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/404tk/cloudtoolkit/pkg/providers/gcp/api"
)

// GetProjectIamPolicy returns the project-level IAM policy via
// cloudresourcemanager projects:getIamPolicy.
func (d *Driver) GetProjectIamPolicy(ctx context.Context, project string) (api.IamPolicy, error) {
	if d == nil || d.Client == nil {
		return api.IamPolicy{}, fmt.Errorf("gcp iam: nil client")
	}
	body, err := json.Marshal(api.GetIamPolicyRequest{
		Options: &api.GetPolicyOptions{RequestedPolicyVersion: 3},
	})
	if err != nil {
		return api.IamPolicy{}, err
	}
	var policy api.IamPolicy
	err = d.Client.Do(ctx, api.Request{
		Method:  http.MethodPost,
		BaseURL: api.ResourceManagerBaseURL,
		Path:    "/v1/projects/" + url.PathEscape(project) + ":getIamPolicy",
		Body:    body,
	}, &policy)
	return policy, err
}

// SetProjectIamPolicy writes a new project policy via
// cloudresourcemanager projects:setIamPolicy. The supplied policy must carry
// the etag from the prior Get to satisfy optimistic concurrency.
func (d *Driver) SetProjectIamPolicy(ctx context.Context, project string, policy api.IamPolicy) (api.IamPolicy, error) {
	if d == nil || d.Client == nil {
		return api.IamPolicy{}, fmt.Errorf("gcp iam: nil client")
	}
	body, err := json.Marshal(api.SetIamPolicyRequest{Policy: policy})
	if err != nil {
		return api.IamPolicy{}, err
	}
	var updated api.IamPolicy
	err = d.Client.Do(ctx, api.Request{
		Method:  http.MethodPost,
		BaseURL: api.ResourceManagerBaseURL,
		Path:    "/v1/projects/" + url.PathEscape(project) + ":setIamPolicy",
		Body:    body,
	}, &updated)
	return updated, err
}

// AddBinding performs a read-modify-write on the project policy to add member
// to role. If the role already exists in the policy, member is appended;
// otherwise a new binding is created.
func (d *Driver) AddBinding(ctx context.Context, project, role, member string) (api.IamPolicy, error) {
	policy, err := d.GetProjectIamPolicy(ctx, project)
	if err != nil {
		return api.IamPolicy{}, err
	}
	policy = mutatePolicy(policy, role, member, true)
	return d.SetProjectIamPolicy(ctx, project, policy)
}

// RemoveBinding performs a read-modify-write on the project policy to remove
// member from role. Empty bindings are pruned.
func (d *Driver) RemoveBinding(ctx context.Context, project, role, member string) (api.IamPolicy, error) {
	policy, err := d.GetProjectIamPolicy(ctx, project)
	if err != nil {
		return api.IamPolicy{}, err
	}
	policy = mutatePolicy(policy, role, member, false)
	return d.SetProjectIamPolicy(ctx, project, policy)
}

// mutatePolicy returns a policy with member added to (or removed from) the
// binding identified by role. The returned policy keeps the original etag so
// the caller can pass it straight to SetProjectIamPolicy.
func mutatePolicy(policy api.IamPolicy, role, member string, add bool) api.IamPolicy {
	role = strings.TrimSpace(role)
	member = strings.TrimSpace(member)
	bindings := append([]api.Binding(nil), policy.Bindings...)
	if add {
		idx := bindingIndex(bindings, role)
		if idx < 0 {
			bindings = append(bindings, api.Binding{Role: role, Members: []string{member}})
		} else {
			if !containsMember(bindings[idx].Members, member) {
				bindings[idx].Members = append(bindings[idx].Members, member)
			}
		}
	} else {
		idx := bindingIndex(bindings, role)
		if idx >= 0 {
			bindings[idx].Members = removeMember(bindings[idx].Members, member)
			if len(bindings[idx].Members) == 0 {
				bindings = append(bindings[:idx], bindings[idx+1:]...)
			}
		}
	}
	return api.IamPolicy{
		Version:  policy.Version,
		Etag:     policy.Etag,
		Bindings: bindings,
	}
}

func bindingIndex(bindings []api.Binding, role string) int {
	for i, b := range bindings {
		if strings.EqualFold(b.Role, role) {
			return i
		}
	}
	return -1
}

func containsMember(members []string, member string) bool {
	for _, m := range members {
		if strings.EqualFold(m, member) {
			return true
		}
	}
	return false
}

func removeMember(members []string, member string) []string {
	out := make([]string, 0, len(members))
	for _, m := range members {
		if !strings.EqualFold(m, member) {
			out = append(out, m)
		}
	}
	return out
}
