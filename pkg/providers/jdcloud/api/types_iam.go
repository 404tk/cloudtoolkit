package api

type DescribeSubUsersResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		SubUsers []SubUser `json:"subUsers"`
		Total    int       `json:"total"`
	} `json:"result"`
}

type DescribeSubUserResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		SubUser SubUser `json:"subUser"`
	} `json:"result"`
}

type SubUser struct {
	Pin        string `json:"pin"`
	Name       string `json:"name"`
	Account    string `json:"account"`
	CreateTime string `json:"createTime"`
}

// CreateSubUserInfo mirrors the SDK's CreateSubUserInfo payload. Only the
// fields relevant to the validation flow are serialised; the rest are omitted
// so JDCloud applies the documented defaults.
type CreateSubUserInfo struct {
	Name              string `json:"name"`
	Password          string `json:"password"`
	ConsoleLogin      *bool  `json:"consoleLogin,omitempty"`
	CreateAk          *bool  `json:"createAk,omitempty"`
	NeedResetPassword *bool  `json:"needResetPassword,omitempty"`
	Description       string `json:"description,omitempty"`
}

type CreateSubUserRequest struct {
	CreateSubUserInfo CreateSubUserInfo `json:"createSubUserInfo"`
}

type CreateSubUserResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		SubUser CreateSubUserResult `json:"subUser"`
	} `json:"result"`
}

type CreateSubUserResult struct {
	Pin          string `json:"pin"`
	Name         string `json:"name"`
	Account      string `json:"account"`
	AccessKey    string `json:"accessKey"`
	SecretKey    string `json:"secretKey"`
	ConsoleLogin *bool  `json:"consoleLogin,omitempty"`
}

type AttachSubUserPolicyRequest struct {
	SubUser        string `json:"subUser"`
	PolicyName     string `json:"policyName"`
	ScopeID        string `json:"scopeId,omitempty"`
	AllowAddPolicy string `json:"allowAddPolicy,omitempty"`
}

type AttachSubUserPolicyResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct{}      `json:"result"`
}

type DetachSubUserPolicyRequest struct {
	SubUser              string `json:"subUser"`
	PolicyName           string `json:"policyName"`
	ScopeID              string `json:"scopeId,omitempty"`
	AllowDetachAddPolicy string `json:"allowDetachAddPolicy,omitempty"`
}

type DetachSubUserPolicyResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct{}      `json:"result"`
}

type DeleteSubUserResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct{}      `json:"result"`
}

// DescribeAttachedPoliciesResponse maps the
// `GET /subUser/{subUser}:describeAttachedPolicies` action that lists managed
// policies bound to a sub user. The exact field names follow the JDCloud SDK
// convention used by sibling :attach/:detach actions.
type DescribeAttachedPoliciesResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		Policies []AttachedPolicy `json:"policies"`
	} `json:"result"`
}

type AttachedPolicy struct {
	PolicyName  string `json:"policyName"`
	PolicyType  string `json:"policyType,omitempty"`
	AttachTime  string `json:"attachTime,omitempty"`
	Description string `json:"description,omitempty"`
}

// DescribeUserPinResponse maps GET /regions/{regionId}/user:describeUserPin.
// When called with master AK/SK the returned pin is the master account's pin,
// which is what the JDCloud sub-account login URL expects.
type DescribeUserPinResponse struct {
	RequestID string        `json:"requestId"`
	Error     *APIErrorBody `json:"error,omitempty"`
	Result    struct {
		Pin string `json:"pin"`
	} `json:"result"`
}
