package api

const ResourceManagerBaseURL = "https://cloudresourcemanager.googleapis.com"

// IamPolicy is the policy document used by both projects:getIamPolicy and
// projects:setIamPolicy. Etag must be round-tripped to detect concurrent
// modifications.
type IamPolicy struct {
	Version  int       `json:"version,omitempty"`
	Etag     string    `json:"etag,omitempty"`
	Bindings []Binding `json:"bindings,omitempty"`
}

// Binding maps a role to one or more members.
type Binding struct {
	Role    string   `json:"role"`
	Members []string `json:"members"`
}

// GetIamPolicyRequest is the body of projects:getIamPolicy. The
// requestedPolicyVersion lets the caller select v3 features (conditions);
// CTK does not use conditions and asks for v1.
type GetIamPolicyRequest struct {
	Options *GetPolicyOptions `json:"options,omitempty"`
}

type GetPolicyOptions struct {
	RequestedPolicyVersion int `json:"requestedPolicyVersion,omitempty"`
}

// SetIamPolicyRequest is the body of projects:setIamPolicy. The Policy
// embeds the etag returned by the previous Get for optimistic concurrency.
type SetIamPolicyRequest struct {
	Policy IamPolicy `json:"policy"`
}
